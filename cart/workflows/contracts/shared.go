package contracts

import (
	"errors"
	"time"

	"go.temporal.io/sdk/workflow"
)

const OrderTaskQueue = "ORDER_TASK_QUEUE"
const OrderAddProductRequestChannel = "ADD_PRODUCT_REQUEST_CHANNEL"
const OrderAddProductResponseChannel = "ADD_PRODUCT_RESPONSE_CHANNEL"
const OrderUpdateProductRequestChannel = "UPDATE_PRODUCT_REQUEST_CHANNEL"
const OrderUpdateProductResponseChannel = "UPDATE_PRODUCT_RESPONSE_CHANNEL"
const OrderPaymentRequestChannel = "PAYMENT_REQUEST_CHANNEL"
const OrderPaymentResponseChannel = "PAYMENT_RESPONSE_CHANNEL"
const OrderPaymentApprovedChannel = "PAYMENT_DONE_REQUEST_CHANNEL"

type To struct {
	WorkflowId  string
	ChannelName string
}

type ReplyTo struct {
	WorkflowId  string
	ChannelName string
}
type Request struct {
	To      To
	ReplyTo ReplyTo
	Body    interface{}
	Timeout time.Duration
}

func NewRequest(to To, reply ReplyTo, body interface{}, timeout time.Duration) Request {
	request := Request{}
	request.To = to
	request.ReplyTo = reply
	request.Timeout = timeout
	request.Body = body
	return request
}

type Response struct {
	Body interface{}
}

func (request *Request) GetData(v interface{}) {
	v = request.Body
}

func (request *Request) Send(ctx workflow.Context, response interface{}) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Sending signal request ", request.To.ChannelName)
	requestReceived := false
	ch := workflow.GetSignalChannel(ctx, request.ReplyTo.ChannelName)
	childCtx, cancel := workflow.WithCancel(ctx)
	s := workflow.NewSelector(ctx)
	s.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &response)
		requestReceived = true
		cancel()
	})
	var err error
	emailTimer := workflow.NewTimer(childCtx, 10*time.Second)
	s.AddFuture(emailTimer, func(f workflow.Future) {
		if !requestReceived {
			logger.Info("Timeout %s", request.ReplyTo.WorkflowId)
			err = errors.New("Timeout")
		}
	})
	logger.Info("Waiting for response %s", request.ReplyTo.WorkflowId)
	err = workflow.SignalExternalWorkflow(
		ctx,
		request.To.WorkflowId,
		"",
		request.To.ChannelName,
		request,
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Workflow Error.", err)
		return err
	}
	s.Select(ctx)
	logger.Info("response came ", response)
	return err
}

func (request *Request) Reply(ctx workflow.Context, response interface{}) error {
	err := workflow.SignalExternalWorkflow(
		ctx,
		request.ReplyTo.WorkflowId,
		"",
		request.ReplyTo.ChannelName,
		response,
	).Get(ctx, nil)
	return err
}
