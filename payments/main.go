package payments

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	Temporal "go.temporal.io/sdk/client"
)

var (
	payments = make(map[string]RegisterPaymentApprovalRequest)
)

type AcceptPaymentContract struct {
	Status string
}

type RegisterPaymentApprovalRequest struct {
	TaskToken string
	CartId    string
}
type PaymentController struct {
	Client Temporal.Client
}

func (controller *PaymentController) PutPayment(c *gin.Context) {
	bodyRaw, _ := ioutil.ReadAll(c.Request.Body)
	var body AcceptPaymentContract
	err := json.Unmarshal(bodyRaw, &body)
	id, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	payment, ok := payments[id.String()]
	if !ok {
		c.JSON(http.StatusNotFound, id.String())
		return
	}

	if err = controller.Client.CompleteActivity(context.Background(), []byte(payment.TaskToken), body.Status, nil); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	delete(payments, id.String())
	c.JSON(http.StatusOK, "OK")
}

func (controller *PaymentController) DeletePayment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	delete(payments, id.String())
	c.JSON(http.StatusOK, "OK")
}
func (controller *PaymentController) PostPayment(c *gin.Context) {
	bodyRaw, _ := ioutil.ReadAll(c.Request.Body)
	var body RegisterPaymentApprovalRequest
	err := json.Unmarshal(bodyRaw, &body)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	body.TaskToken, err = url.QueryUnescape(body.TaskToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	payments[body.CartId] = body
	c.JSON(http.StatusOK, "OK")
	return
}

func (controller *PaymentController) GetPayment(c *gin.Context) {
	c.JSON(http.StatusOK, payments)
	return
}

func (controller *PaymentController) v1(router *gin.RouterGroup) {
	v1 := router.Group("/v1")
	v1.PUT("/:paymentId/", controller.PutPayment)
	v1.POST("/", controller.PostPayment)
	v1.GET("/", controller.GetPayment)
	v1.DELETE("/:paymentId/", controller.DeletePayment)
}

func RegisterRoutes(rg *gin.Engine, client *Temporal.Client) {
	controller := PaymentController{Client: *client}
	payments := rg.Group("/payments")
	controller.v1(payments)
}
