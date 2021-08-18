// @@@SNIPSTART hello-world-project-template-go-worker
package main

import (
	"log"

	"services/order"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Create the client object just once per process
	c, err := client.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()
	// This worker hosts both Worker and Activity functions
	w := worker.New(c, order.OrderTaskQueue, worker.Options{})
	w.RegisterWorkflow(order.OrderWorkflow)
	//w.RegisterActivity(order.ComposeGreeting)

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
		log.Fatalln("unable to start Worker", err)
	}
}

// @@@SNIPEND
