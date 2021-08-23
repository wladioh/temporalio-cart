package workflows

import (
	"services/cart/workflows/contracts"
	"services/cart/workflows/requests"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type Cart struct {
	context     workflow.Context
	paying      bool
	canceled    bool
	timeout     bool
	hasProducts bool
	emailSended bool
	id          uuid.UUID
}

func (cart *Cart) AddProduct(c workflow.ReceiveChannel, more bool) {
	var request contracts.Request
	c.Receive(cart.context, &request)
	var data requests.UpdateProductRequest
	request.GetData(&data)
	logger := workflow.GetLogger(cart.context)

	logger.Info("AddProduct ", data.ProductId)

	logger.Info("Sending response", request.CallingWorkflowId)

	err := workflow.SignalExternalWorkflow(
		cart.context,
		request.CallingWorkflowId,
		"",
		contracts.OrderAddProductResponseChannel,
		requests.AddProductResponse{Status: "OK"},
	).Get(cart.context, nil)
	cart.hasProducts = true
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.CallingWorkflowId)
	}
}

func (order *Cart) UpdateProduct(c workflow.ReceiveChannel, more bool) {
	logger := workflow.GetLogger(order.context)
	var request contracts.Request
	c.Receive(order.context, &request)
	var data requests.UpdateProductRequest
	request.GetData(&data)
	logger.Info("Update Product ", data.ProductId)

	logger.Info("Sending response", request.CallingWorkflowId)
	err := workflow.SignalExternalWorkflow(
		order.context,
		request.CallingWorkflowId,
		"",
		contracts.OrderUpdateProductResponseChannel,
		requests.UpdateProductResponse{Status: "OK"},
	).Get(order.context, nil)
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.CallingWorkflowId)
	}
}

func (cart *Cart) Payment(c workflow.ReceiveChannel, more bool) {
	logger := workflow.GetLogger(cart.context)

	logger.Info("Start Payment Order ")
	var request contracts.Request
	c.Receive(cart.context, &request)
	var data requests.PaymentRequest
	request.GetData(&data)
	cart.paying = true
	logger.Info("Pay Order ", data.OrderId)
	logger.Info("Sending response", request.CallingWorkflowId)
	err := workflow.SignalExternalWorkflow(
		cart.context,
		request.CallingWorkflowId,
		"",
		contracts.OrderPaymentResponseChannel,
		requests.PaymentResponse{Status: "OK"},
	).Get(cart.context, nil)
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.CallingWorkflowId)
	}
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
		StartToCloseTimeout: time.Second * 5,
	}
	cart := Cart{context: ctx, id: id}
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, options)
	options = workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 10 * time.Second,
	}
	waitTimeForCancelCart := 1 * time.Minute
	waitTimeForSendEmail := 30 * time.Second
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
			cancelCartTimer := workflow.NewTimer(childCtx, waitTimeForCancelCart)
			s.AddFuture(cancelCartTimer, func(f workflow.Future) {
				if requestReceived {
					cart.emailSended = false
					return
				}
				logger.Info("Canceling this cart.")
				cart.canceled = true
			})
		} else {
			emailTimer := workflow.NewTimer(childCtx, waitTimeForSendEmail)
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
			break
		}
	}
	logger.Info("Workflow completed.")
	return nil
}
