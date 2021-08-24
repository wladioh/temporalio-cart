package requests

import (
	"errors"
	"time"

	"services/cart/workflows/contracts"

	"go.temporal.io/sdk/workflow"
)

type UpdateProductRequest struct {
	OrderId   string
	ProductId string
	Quantity  int
}

type UpdateProductResponse struct {
	Error  string
	Status string
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
	wrapRequest := contracts.Request{
		Body: request,
		To: contracts.To{
			WorkflowId:  request.OrderId,
			ChannelName: contracts.OrderUpdateProductRequestChannel,
		},
		ReplyTo: contracts.ReplyTo{
			WorkflowId:  workflowID,
			ChannelName: contracts.OrderUpdateProductResponseChannel,
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
