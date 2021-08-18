package main

import (
	"services/cart"
	"services/coupon"
	"services/payments"
	"services/shipment"
	"services/stock"

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
	router.Run(":5000")
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
