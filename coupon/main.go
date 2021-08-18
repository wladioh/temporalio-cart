package coupon

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func v1(router *gin.RouterGroup) {
	v1 := router.Group("/v1")

	v1.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, "users")
	})
	v1.GET("/comments", func(c *gin.Context) {
		c.JSON(http.StatusOK, "users comments")
	})
	v1.GET("/pictures", func(c *gin.Context) {
		c.JSON(http.StatusOK, "users pictures")
	})
}

func RegisterRoutes(rg *gin.Engine) {
	shipping := rg.Group("/coupon")
	v1(shipping)
}
