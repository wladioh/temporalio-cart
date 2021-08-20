package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"services/cart"
	cartWorkflows "services/cart/workflows"
	"services/coupon"
	"services/payments"
	"services/shipment"
	"services/stock"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	router *gin.Engine
)

// Run will start the server
func main() {
	router = gin.Default()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	registerRoutes()
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
func registerRoutes() {
	payments.RegisterRoutes(router)
	shipment.RegisterRoutes(router)
	stock.RegisterRoutes(router)
	cart.RegisterRoutes(router)
	coupon.RegisterRoutes(router)
}
