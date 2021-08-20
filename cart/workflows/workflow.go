package workflows

import (
	"services/cart/workflows/contracts"
	"services/cart/workflows/requests"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type Order struct {
	context workflow.Context
	paid    bool
	id      uuid.UUID
}

func (order *Order) AddProduct(c workflow.ReceiveChannel, more bool) {
	var request contracts.Request
	c.Receive(order.context, &request)
	var data requests.UpdateProductRequest
	request.GetData(&data)
	logger := workflow.GetLogger(order.context)

	logger.Info("AddProduct ", data.ProductId)

	logger.Info("Sending response", request.CallingWorkflowId)

	err := workflow.SignalExternalWorkflow(
		order.context,
		request.CallingWorkflowId,
		"",
		contracts.OrderAddProductResponseChannel,
		requests.AddProductResponse{Status: "OK"},
	).Get(order.context, nil)
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.CallingWorkflowId)
	}
}

func (order *Order) UpdateProduct(c workflow.ReceiveChannel, more bool) {
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

func (order *Order) Pay(c workflow.ReceiveChannel, more bool) {
	logger := workflow.GetLogger(order.context)

	logger.Info("Start Payment Order ")
	var request contracts.Request
	c.Receive(order.context, &request)
	var data requests.PaymentRequest
	request.GetData(&data)
	order.paid = true
	logger.Info("Pay Order ", data.OrderId)
	logger.Info("Sending response", request.CallingWorkflowId)
	err := workflow.SignalExternalWorkflow(
		order.context,
		request.CallingWorkflowId,
		"",
		contracts.OrderPaymentResponseChannel,
		requests.PaymentResponse{Status: "OK"},
	).Get(order.context, nil)
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.CallingWorkflowId)
	}
}

func OrderWorkflow(ctx workflow.Context, id uuid.UUID) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 5,
	}
	order := Order{context: ctx, id: id}
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, options)
	options = workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 10 * time.Second,
	}
	addProductChan := workflow.GetSignalChannel(ctx, contracts.OrderAddProductRequestChannel)
	updateProductChan := workflow.GetSignalChannel(ctx, contracts.OrderUpdateProductRequestChannel)
	payChan := workflow.GetSignalChannel(ctx, contracts.OrderPaymentRequestChannel)
	s := workflow.NewSelector(ctx)
	timerFuture := workflow.NewTimer(ctx, 24*time.Hour)
	s.AddFuture(timerFuture, func(f workflow.Future) {
		if !order.paid { // processing is not done yet when timer fires, send notification email
			// _ = workflow.ExecuteActivity(ctx, SendEmailActivity).Get(ctx, nil)
			logger.Info("Send email remembering about this cart.")
		}
	})
	for {
		s.AddReceive(addProductChan, order.AddProduct)
		s.AddReceive(updateProductChan, order.UpdateProduct)
		s.AddReceive(payChan, order.Pay)
		s.Select(ctx)
		if order.paid {
			logger.Info("Pay")
			break
		}
	}
	logger.Info("Workflow completed.")
	return nil
}
