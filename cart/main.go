package cart

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"services/cart/workflows"
	"services/cart/workflows/contracts"
	"services/cart/workflows/requests"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	Temporal "go.temporal.io/sdk/client"
)

func GetData(c *gin.Context, v interface{}) error {
	bodyRaw, _ := ioutil.ReadAll(c.Request.Body)

	err := json.Unmarshal(bodyRaw, v)
	return err
}

type CartController struct {
	Client Temporal.Client
}

func (controller *CartController) CreateCart(c *gin.Context) {
	id := uuid.New()
	options := Temporal.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: contracts.OrderTaskQueue,
	}
	we, err := controller.Client.ExecuteWorkflow(context.Background(), options, workflows.OrderWorkflow, id)
	if err != nil {
		log.Fatalln("unable to complete Workflow", err)
	}
	log.Print("Order Started", we.GetID())
	c.JSON(http.StatusOK, id)
}

func (controller *CartController) AddProduct(c *gin.Context) {
	var request requests.AddProductRequest
	err := GetData(c, &request)
	if err != nil {
		c.String(http.StatusBadRequest, "Hello %s ", err)
		return
	}
	id, _ := uuid.Parse(c.Param("cartId"))

	var status string
	request.OrderId = id.String()
	err = controller.ExecuteWorkflow(requests.AddProductRequestWorkflow, request, &status)
	if err != nil {
		log.Print("Error signaling client", err)
		c.String(http.StatusBadRequest, "%s ", err)
		return
	}

	c.JSON(http.StatusOK, status)
}

func (controller *CartController) ExecuteWorkflow(workflow interface{}, request interface{}, response interface{}) error {
	options := Temporal.StartWorkflowOptions{
		ID:        uuid.NewString(),
		TaskQueue: contracts.OrderTaskQueue,
	}
	log.Printf("Starting workflow %s", options.ID)
	ctx := context.Background()
	we, err := controller.Client.ExecuteWorkflow(ctx, options, workflow, request)
	if err != nil {
		return err
	}
	log.Printf("awating response %s", options.ID)
	err = we.Get(ctx, &response)
	return err
}

func (controller *CartController) UpdateProduct(c *gin.Context) {
	var request requests.UpdateProductRequest
	err := GetData(c, &request)
	if err != nil {
		c.String(http.StatusBadRequest, "Hello %s ", err)
		return
	}

	id, _ := uuid.Parse(c.Param("cartId"))

	var status string
	request.OrderId = id.String()
	err = controller.ExecuteWorkflow(requests.UpdateProductRequestWorkflow, request, &status)
	if err != nil {
		log.Print("Error signaling client", err)
		c.String(http.StatusBadRequest, "%s ", err)
		return
	}

	c.JSON(http.StatusOK, status)
}

func (controller *CartController) Payment(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("cartId"))
	var status string
	request := requests.PaymentRequest{OrderId: id.String()}
	err := controller.ExecuteWorkflow(requests.PaymentRequestWorkflow, request, &status)
	if err != nil {
		c.String(http.StatusBadRequest, "Error %s", err)
	}
	c.JSON(http.StatusOK, status)
}

func v1(router *gin.RouterGroup) {
	cl, err := Temporal.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	// defer cl.Close()
	controller := CartController{Client: cl}
	v1 := router.Group("/v1")
	v1.POST("/", controller.CreateCart)
	v1.POST("/:cartId/product", controller.AddProduct)
	v1.PUT("/:cartId/product", controller.UpdateProduct)
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
	v1.POST("/:cartId/payment", controller.Payment)
	v1.POST("/:cartId/payment/method", func(c *gin.Context) {
		id := c.Param("cartId")
		c.String(http.StatusOK, "Hello %s ", id)
	})
}

func RegisterRoutes(rg *gin.Engine) {
	shipping := rg.Group("/cart")
	v1(shipping)
}
