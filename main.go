package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"services/cart"
	cartWorkflows "services/cart/workflows"
	propagation "services/cart/workflows/propagator"
	"services/coupon"
	"services/payments"
	"services/shipment"
	"services/stock"
	zapadapter "services/zap"
	"time"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
	Temporal "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

var (
	router *gin.Engine
)

// Run will start the server
func main() {
	cl, err := Temporal.NewClient(client.Options{
		HostPort: client.DefaultHostPort,
		// Tracer:             opentracing.GlobalTracer(),
		Logger:             zapadapter.NewZapAdapter(zapadapter.NewZapLogger()),
		ContextPropagators: []workflow.ContextPropagator{propagation.NewContextPropagator()},
	})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer cl.Close()
	router = gin.Default()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	registerRoutes(&cl)
	srv := &http.Server{
		Addr:    ":5000",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	go func() {
		if err := cartWorkflows.Worker(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}

// registerRoutes will create our routes of our entire application
// this way every group of routes can be defined in their own file
// so this one won't be so messy
func registerRoutes(client *Temporal.Client) {
	payments.RegisterRoutes(router, client)
	shipment.RegisterRoutes(router)
	stock.RegisterRoutes(router)
	cart.RegisterRoutes(router, client)
	coupon.RegisterRoutes(router)
}
