// @@@SNIPSTART hello-world-project-template-go-workflow
package order

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type AddProductRequest struct {
	ProductId string
	Quantity  int
}

type Order struct {
	context workflow.Context
	paid    bool
	id      uuid.UUID
}

func (order *Order) AddProduct(c workflow.ReceiveChannel, more bool) {
	var signalVal AddProductRequest
	c.Receive(order.context, &signalVal)
	workflow.GetLogger(order.context).Info("AddProduct ", signalVal.ProductId)
}

func (order *Order) UpdateProduct(c workflow.ReceiveChannel, more bool) {
	var signalVal AddProductRequest
	c.Receive(order.context, &signalVal)
	workflow.GetLogger(order.context).Info("Update Product ", signalVal.ProductId)
}

func (order *Order) Pay(c workflow.ReceiveChannel, more bool) {
	var signalVal string
	c.Receive(order.context, &signalVal)
	order.paid = true
	workflow.GetLogger(order.context).Info("Pay Order ", signalVal)
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
	addProductChan := workflow.GetSignalChannel(ctx, OrderAddProductChannel)
	updateProductChan := workflow.GetSignalChannel(ctx, OrderUpdateProductChannel)
	payChan := workflow.GetSignalChannel(ctx, OrderPayChannel)
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

// @@@SNIPEND
