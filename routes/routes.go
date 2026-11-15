package routes

import (
	"quotation/controllers"
	"quotation/web"

	"github.com/gin-gonic/gin"
)

func New(customers *controllers.CustomerController, quotations *controllers.QuotationController) (*gin.Engine, error) {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	web.Register(router)

	api := router.Group("/api/v1")
	customerRoutes := api.Group("/customers")
	{
		customerRoutes.POST("", customers.Create)
		customerRoutes.GET("", customers.List)
		customerRoutes.GET("/:id", customers.Get)
		customerRoutes.PUT("/:id", customers.Update)
		customerRoutes.DELETE("/:id", customers.Delete)
	}
	quotationRoutes := api.Group("/quotations")
	{
		quotationRoutes.POST("", quotations.Create)
		quotationRoutes.GET("", quotations.List)
		quotationRoutes.GET("/:id", quotations.Get)
		quotationRoutes.PUT("/:id", quotations.Update)
		quotationRoutes.PATCH("/:id/status", quotations.UpdateStatus)
		quotationRoutes.DELETE("/:id", quotations.Delete)
		quotationRoutes.GET("/:id/pdf", quotations.PDF)
	}
	return router, nil
}
