package requests

import (
	"errors"
	"services/cart/workflows/contracts"
	"time"

	"go.temporal.io/sdk/workflow"
)

type AddProductRequest struct {
	ProductId string
	Quantity  int
	OrderId   string
}

type AddProductResponse struct {
	Error  string
	Status string
}

func AddProductRequestWorkflow(ctx workflow.Context, request AddProductRequest) (string, error) {
	options := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		// StartToCloseTimeout:    10 * time.Second,
		// ScheduleToCloseTimeout: 20 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	wrapRequest := contracts.NewRequest(
		contracts.To{
			WorkflowId:  request.OrderId,
			ChannelName: contracts.OrderAddProductRequestChannel,
		},
		contracts.ReplyTo{
			WorkflowId:  workflowID,
			ChannelName: contracts.OrderAddProductResponseChannel,
		},
		request,
		11*time.Second,
	)
	var res AddProductResponse
	err := wrapRequest.Send(ctx, &res)

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
