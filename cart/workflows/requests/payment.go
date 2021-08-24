package requests

import (
	"errors"
	"time"

	"services/cart/workflows/contracts"

	"go.temporal.io/sdk/workflow"
)

type PaymentRequest struct {
	OrderId string
}

type PaymentResponse struct {
	Error  string
	Status string
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
	wrapRequest := contracts.Request{
		Body: request,
		To: contracts.To{
			WorkflowId:  request.OrderId,
			ChannelName: contracts.OrderPaymentRequestChannel,
		},
		ReplyTo: contracts.ReplyTo{
			WorkflowId:  workflowID,
			ChannelName: contracts.OrderPaymentResponseChannel,
		},
	}
	err := wrapRequest.Send(ctx, &res)

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
