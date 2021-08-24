package workflows

import (
	"services/cart/workflows/contracts"
	"services/cart/workflows/requests"
	"time"

	propagation "services/cart/workflows/propagator"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

const PAYMENT_RETRY_TIME = 10 * time.Second
const PAYMENT_TIMEOUT_BREAK = 20 * time.Second
const CART_ADANDONED_AFTER_EMAIL = 1 * time.Minute
const CART_ADANDONED_TIME = 30 * time.Second

type Cart struct {
	context     workflow.Context
	paying      bool
	canceled    bool
	hasProducts bool
	emailSended bool
	id          uuid.UUID
}

func (cart *Cart) AddProduct(c workflow.ReceiveChannel, more bool) {
	var data requests.AddProductRequest
	request := cart.WaitRequest(c, &data)
	logger := workflow.GetLogger(cart.context)

	logger.Info("AddProduct ", data.ProductId)
	var values propagation.Values
	if err := workflow.ExecuteActivity(cart.context, SampleActivity).Get(cart.context, &values); err != nil {
		logger.Error("Workflow failed.", "Error", err)
	}
	logger.Info("context propagated to activity.", values.Key, values.Value)
	logger.Info("Sending response", request.ReplyTo.WorkflowId)

	err := request.Reply(cart.context, requests.AddProductResponse{Status: "OK"})
	cart.hasProducts = true
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.ReplyTo.WorkflowId)
	}
}

func (order *Cart) WaitRequest(c workflow.ReceiveChannel, data interface{}) contracts.Request {
	var request contracts.Request
	c.Receive(order.context, &request)
	request.GetData(&data)
	return request
}

func (order *Cart) UpdateProduct(c workflow.ReceiveChannel, more bool) {
	logger := workflow.GetLogger(order.context)
	var data requests.UpdateProductRequest
	request := order.WaitRequest(c, &data)
	logger.Info("Update Product ", data.ProductId)
	logger.Info("Sending response", request.ReplyTo.WorkflowId)
	err := request.Reply(order.context, requests.UpdateProductResponse{Status: "OK"})
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.ReplyTo.WorkflowId)
	}
}

func (cart *Cart) Payment(c workflow.ReceiveChannel, more bool) {
	logger := workflow.GetLogger(cart.context)
	logger.Info("Start Payment Cart")
	var data requests.PaymentRequest
	request := cart.WaitRequest(c, &data)
	cart.paying = true
	logger.Info("Paying Cart", data.OrderId)
	logger.Info("Sending response", request.ReplyTo.ChannelName)
	err := request.Reply(cart.context, requests.PaymentResponse{Status: "OK"})
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.ReplyTo.ChannelName)
	}
}

func (cart *Cart) AwaitPaymentApproval() (string, error) {
	var a *Activities
	var status string
	logger := workflow.GetLogger(cart.context)
	logger.Info("Start Payment")
	options := workflow.ActivityOptions{
		StartToCloseTimeout:    PAYMENT_RETRY_TIME,    // TIMEOUT TO RETRY
		ScheduleToCloseTimeout: PAYMENT_TIMEOUT_BREAK, // TIMEOUT TO BREAK
	}
	ctx := workflow.WithActivityOptions(cart.context, options)
	err := workflow.ExecuteActivity(ctx, a.WaitForPaymentActivity, cart.id).Get(ctx, &status)
	if err != nil {
		logger.Error("Workflow Error. %s", err)
		_ = workflow.ExecuteActivity(ctx, a.RemovePaymentActivity, cart.id).Get(ctx, &status)
		return "", err
	}
	logger.Info("Cart ", "status", status)
	return status, err
}

func (cart *Cart) CalledMidware(called *bool, cancel workflow.CancelFunc, f func(c workflow.ReceiveChannel, more bool)) func(c workflow.ReceiveChannel, more bool) {
	return func(c workflow.ReceiveChannel, more bool) {
		*(called) = true
		cancel()
		f(c, more)
	}
}

func CartWorkflow(ctx workflow.Context, id uuid.UUID) error {
	options := workflow.ActivityOptions{
		// ScheduleToStartTimeout: 10 * time.Minute,
		// StartToCloseTimeout:    10 * time.Second,
		StartToCloseTimeout: 20 * time.Second,
	}
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, options)
	cart := Cart{context: ctx, id: id}
	cando := []string{"ADD_PRODUCT", "UPDATE_PRODUCT", "PAY"}

	_ = workflow.SetQueryHandler(ctx, "cando", func(input []byte) ([]string, error) {
		return cando, nil
	})
	addProductChan := workflow.GetSignalChannel(ctx, contracts.OrderAddProductRequestChannel)
	updateProductChan := workflow.GetSignalChannel(ctx, contracts.OrderUpdateProductRequestChannel)
	payChan := workflow.GetSignalChannel(ctx, contracts.OrderPaymentRequestChannel)
	cart.emailSended = false
	for {
		requestReceived := false
		s := workflow.NewSelector(ctx)
		childCtx, cancel := workflow.WithCancel(ctx)
		s.AddReceive(addProductChan, cart.CalledMidware(&requestReceived, cancel, cart.AddProduct))
		s.AddReceive(updateProductChan, cart.CalledMidware(&requestReceived, cancel, cart.UpdateProduct))
		s.AddReceive(payChan, cart.CalledMidware(&requestReceived, cancel, cart.Payment))
		if cart.emailSended {
			cancelCartTimer := workflow.NewTimer(childCtx, CART_ADANDONED_AFTER_EMAIL)
			s.AddFuture(cancelCartTimer, func(f workflow.Future) {
				if requestReceived {
					cart.emailSended = false
					return
				}
				logger.Info("Canceling this cart.")
				cart.canceled = true
			})
		} else {
			emailTimer := workflow.NewTimer(childCtx, CART_ADANDONED_TIME)
			s.AddFuture(emailTimer, func(f workflow.Future) {
				if requestReceived {
					cart.emailSended = false
					return
				}
				if cart.hasProducts {
					cart.emailSended = true
					logger.Info("Send email remembering about this cart.")
				} else {
					cart.canceled = true
					logger.Info("Canceling this cart.")
				}
			})
		}
		s.Select(ctx)
		if cart.paying || cart.canceled {
			logger.Info("Pay")
			cando = make([]string, 0)
			break
		}
	}
	status, err := cart.AwaitPaymentApproval()
	if err != nil {
		logger.Error("Erro on payment")
		return nil
	}
	logger.Info("Workflow completed. ", "status", status)
	return nil
}
