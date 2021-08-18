package cart

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"services/order"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

func GetData(c *gin.Context, v *interface{}) error {
	bodyRaw, _ := ioutil.ReadAll(c.Request.Body)

	err := json.Unmarshal(bodyRaw, v)
	return err
}

func v1(router *gin.RouterGroup) {
	v1 := router.Group("/v1")
	v1.POST("/", func(c *gin.Context) {
		cl, err := client.NewClient(client.Options{})
		if err != nil {
			log.Fatalln("unable to create Temporal client", err)
		}
		defer cl.Close()
		var id = uuid.New()
		options := client.StartWorkflowOptions{
			ID:        id.String(),
			TaskQueue: order.OrderTaskQueue,
		}
		we, err := cl.ExecuteWorkflow(context.Background(), options, order.OrderWorkflow, id)
		if err != nil {
			log.Fatalln("unable to complete Workflow", err)
		}
		log.Print("Order Started", we.GetID())
		c.String(http.StatusOK, "OrderId %s", id)

	})
	v1.POST("/:cartId/product", func(c *gin.Context) {
		var productRequest order.AddProductRequest
		bodyRaw, _ := ioutil.ReadAll(c.Request.Body)

		err := json.Unmarshal(bodyRaw, &productRequest)
		if err != nil {
			c.String(http.StatusBadRequest, "Hello %s ", err)
			return
		}
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s %s", id, productRequest)

		cl, err := client.NewClient(client.Options{})
		if err != nil {
			log.Fatalln("unable to create Temporal client", err)
		}
		defer cl.Close()

		err = cl.SignalWorkflow(context.Background(), id, "", order.OrderAddProductChannel, productRequest)
		if err != nil {
			log.Fatalln("Error signaling client", err)
		}
	})
	v1.PUT("/:cartId/product", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})
	v1.DELETE("/:cartId/product", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})
	v1.POST("/:cartId/coupon", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})
	v1.DELETE("/:cartId/coupon", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})
	v1.POST("/:cartId/address", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})
	v1.POST("/:cartId/payment", func(c *gin.Context) {
		id := c.Param("cartId")
		var productRequest order.AddProductRequest
		bodyRaw, _ := ioutil.ReadAll(c.Request.Body)

		err := json.Unmarshal(bodyRaw, &productRequest)
		if err != nil {
			c.String(http.StatusBadRequest, "Hello %s ", err)
			return
		}

		cl, err := client.NewClient(client.Options{})
		if err != nil {
			log.Fatalln("unable to create Temporal client", err)
		}
		defer cl.Close()

		err = cl.SignalWorkflow(context.Background(), id, "", order.OrderPayChannel, "PAY")
		if err != nil {
			log.Fatalln("Error signaling client", err)
			c.String(http.StatusBadRequest, "Error signaling client", err)
		}
		c.String(http.StatusOK, "Paid")
	})
	v1.POST("/:cartId/payment/method", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})

}

func RegisterRoutes(rg *gin.Engine) {
	shipping := rg.Group("/cart")
	v1(shipping)
}
