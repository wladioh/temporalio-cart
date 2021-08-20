package contracts

import "go.temporal.io/sdk/workflow"

const OrderTaskQueue = "ORDER_TASK_QUEUE"
const OrderAddProductRequestChannel = "ADD_PRODUCT_REQUEST_CHANNEL"
const OrderAddProductResponseChannel = "ADD_PRODUCT_RESPONSE_CHANNEL"
const OrderUpdateProductRequestChannel = "UPDATE_PRODUCT_REQUEST_CHANNEL"
const OrderUpdateProductResponseChannel = "UPDATE_PRODUCT_RESPONSE_CHANNEL"
const OrderPaymentRequestChannel = "PAYMENT_REQUEST_CHANNEL"
const OrderPaymentResponseChannel = "PAYMENT_RESPONSE_CHANNEL"

type Request struct {
	CallingWorkflowId string
	Data              interface{}
}

func (request *Request) GetData(v interface{}) {
	v = request.Data
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
