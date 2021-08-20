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
		ScheduleToStartTimeout: 10 * time.Second,
		StartToCloseTimeout:    10 * time.Second,
		ScheduleToCloseTimeout: 20 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	wrapRequest := contracts.Request{
		CallingWorkflowId: workflowID,
		Data:              request,
	}
	var res AddProductResponse
	err := contracts.SendRequest(ctx, contracts.OrderAddProductRequestChannel, request.OrderId, contracts.OrderAddProductResponseChannel, wrapRequest, &res)

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
