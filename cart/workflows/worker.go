package workflows

import (
	"services/cart/workflows/contracts"
	"services/cart/workflows/requests"

	"github.com/go-resty/resty/v2"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func Worker() error {
	// Create the client object just once per process
	c, err := client.NewClient(client.Options{})
	if err != nil {
		return err
	}
	defer c.Close()
	// This worker hosts both Worker and Activity functions
	w := worker.New(c, contracts.OrderTaskQueue, worker.Options{})
	w.RegisterWorkflow(CartWorkflow)
	w.RegisterWorkflow(requests.AddProductRequestWorkflow)
	w.RegisterWorkflow(requests.UpdateProductRequestWorkflow)
	w.RegisterWorkflow(requests.PaymentRequestWorkflow)
	w.RegisterActivity(SampleActivity)
	client := resty.New().
		SetHostURL("http://localhost:5000").
		EnableTrace().
		SetHeader("Content-Type", "application/json")

	activities := &Activities{
		HttpClient: *client,
	}
	w.RegisterActivity(activities)

	// client := resty.New().
	// 	SetHostURL("https://ab449f68-fa4b-4445-ba51-2cee2ad19ca8.mock.pstmn.io").
	// 	EnableTrace().
	// 	SetHeader("Content-Type", "application/json")

	// activities := &app.Activities{
	// 	HttpClient: *client,
	// }
	//	w.RegisterActivity(activities)

	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		return err
	}
	return nil
}
