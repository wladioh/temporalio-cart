// @@@SNIPSTART hello-world-project-template-go-workflow
package order

import (
	"time"

	"errors"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type Request struct {
	CallingWorkflowId string
	Data              interface{}
}

type AddProductRequest struct {
	ProductId string
	Quantity  int
	OrderId   string
}

type AddProductResponse struct {
	Error  string
	Status string
}

type UpdateProductRequest struct {
	OrderId   string
	ProductId string
	Quantity  int
}

type UpdateProductResponse struct {
	Error  string
	Status string
}
type Order struct {
	context workflow.Context
	paid    bool
	id      uuid.UUID
}
type PaymentRequest struct {
	OrderId string
}
type PaymentResponse struct {
	Error  string
	Status string
}

func (request *Request) GetData(v interface{}) {
	v = request.Data
}

func (order *Order) AddProduct(c workflow.ReceiveChannel, more bool) {
	var request Request
	c.Receive(order.context, &request)
	var data UpdateProductRequest
	request.GetData(&data)
	logger := workflow.GetLogger(order.context)

	logger.Info("AddProduct ", data.ProductId)

	logger.Info("Sending response", request.CallingWorkflowId)

	err := workflow.SignalExternalWorkflow(
		order.context,
		request.CallingWorkflowId,
		"",
		OrderAddProductResponseChannel,
		AddProductResponse{Status: "OK"},
	).Get(order.context, nil)
	if err != nil {
		logger.Error("Sending response return error: ", err)
	} else {
		logger.Info("Send response with success", request.CallingWorkflowId)
	}
}

func (order *Order) UpdateProduct(c workflow.ReceiveChannel, more bool) {
	logger := workflow.GetLogger(order.context)
	var request Request
	c.Receive(order.context, &request)
	var data UpdateProductRequest
	request.GetData(&data)
	logger.Info("Update Product ", data.ProductId)

	logger.Info("Sending response", request.CallingWorkflowId)
	err := workflow.SignalExternalWorkflow(
		order.context,
		request.CallingWorkflowId,
		"",
		OrderUpdateProductResponseChannel,
		UpdateProductResponse{Status: "OK"},
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
	var request Request
	c.Receive(order.context, &request)
	var data PaymentRequest
	request.GetData(&data)
	order.paid = true
	logger.Info("Pay Order ", data.OrderId)
	logger.Info("Sending response", request.CallingWorkflowId)
	err := workflow.SignalExternalWorkflow(
		order.context,
		request.CallingWorkflowId,
		"",
		OrderPaymentResponseChannel,
		PaymentResponse{Status: "OK"},
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
	addProductChan := workflow.GetSignalChannel(ctx, OrderAddProductRequestChannel)
	updateProductChan := workflow.GetSignalChannel(ctx, OrderUpdateProductRequestChannel)
	payChan := workflow.GetSignalChannel(ctx, OrderPaymentRequestChannel)
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

func SendRequest(ctx workflow.Context, channelRequest string, workflowId string, channelResponse string, request interface{}, response interface{}) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Sending signal request ", channelRequest)
	err := workflow.SignalExternalWorkflow(
		ctx,
		workflowId,
		"",
		channelRequest,
		request,
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Workflow Error.", err)
		return err
	}

	ch := workflow.GetSignalChannel(ctx, channelResponse)

	logger.Info("Waiting for response %s", channelResponse)

	ch.Receive(ctx, &response)

	logger.Info("response came ", response)
	return nil
}

func AddProductRequestWorkflow(ctx workflow.Context, request AddProductRequest) (string, error) {
	options := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Second,
		StartToCloseTimeout:    10 * time.Second,
		ScheduleToCloseTimeout: 20 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	wrapRequest := Request{
		CallingWorkflowId: workflowID,
		Data:              request,
	}
	var res AddProductResponse
	err := SendRequest(ctx, OrderAddProductRequestChannel, request.OrderId, OrderAddProductResponseChannel, wrapRequest, &res)

	logger.Info("Workflow completed.")
	if err != nil {
		logger.Error("Workflow Error.", err)
		return "", err
	}
	if res.Error != "" {
		return "", errors.New(res.Error)
	}
	return res.Status, nil
}

func UpdateProductRequestWorkflow(ctx workflow.Context, request UpdateProductRequest) (string, error) {
	options := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Second,
		StartToCloseTimeout:    10 * time.Second,
		ScheduleToCloseTimeout: 20 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	var res UpdateProductResponse
	wrapRequest := Request{
		CallingWorkflowId: workflowID,
		Data:              request,
	}
	err := SendRequest(ctx, OrderUpdateProductRequestChannel, request.OrderId, OrderUpdateProductResponseChannel, wrapRequest, &res)

	if err != nil {
		logger.Error("Workflow Error: ", err)
		return "", err
	}
	if res.Error != "" {
		logger.Error("Workflow Error: ", res.Error)
		return "", errors.New(res.Error)
	}
	logger.Info("Workflow completed.")
	return res.Status, nil
}

func PaymentRequestWorkflow(ctx workflow.Context, request PaymentRequest) (string, error) {
	options := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Second,
		StartToCloseTimeout:    10 * time.Second,
		ScheduleToCloseTimeout: 20 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	var res PaymentResponse
	wrapRequest := Request{
		CallingWorkflowId: workflowID,
		Data:              request,
	}
	err := SendRequest(ctx, OrderPaymentRequestChannel, request.OrderId, OrderPaymentResponseChannel, wrapRequest, &res)

	if err != nil {
		logger.Error("Workflow Error: ", err)
		return "", err
	}
	if res.Error != "" {
		logger.Error("Workflow Error: ", res.Error)
		return "", errors.New(res.Error)
	}
	logger.Info("Workflow completed.")
	return res.Status, nil
}

// @@@SNIPEND
