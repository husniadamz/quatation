package routes

import (
	"quotation/controllers"
	"quotation/middleware"
	"quotation/services"
	"quotation/web"

	"github.com/gin-gonic/gin"
)

func New(customers *controllers.CustomerController, quotations *controllers.QuotationController,
	auth *controllers.AuthController, dashboard *controllers.DashboardController,
	authService services.AuthService) (*gin.Engine, error) {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	web.Register(router)

	api := router.Group("/api/v1")
	api.GET("/auth/setup", auth.SetupStatus)
	api.POST("/auth/setup", auth.Setup)
	api.POST("/auth/login", auth.Login)
	protected := api.Group("")
	protected.Use(middleware.Authenticate(authService))
	protected.GET("/auth/me", auth.Me)
	protected.POST("/auth/logout", auth.Logout)
	protected.GET("/dashboard", dashboard.Get)
	customerRoutes := protected.Group("/customers")
	{
		customerRoutes.POST("", customers.Create)
		customerRoutes.GET("", customers.List)
		customerRoutes.GET("/:id", customers.Get)
		customerRoutes.PUT("/:id", customers.Update)
		customerRoutes.DELETE("/:id", customers.Delete)
	}
	quotationRoutes := protected.Group("/quotations")
	{
		quotationRoutes.POST("", quotations.Create)
		quotationRoutes.GET("", quotations.List)
		quotationRoutes.GET("/:id", quotations.Get)
		quotationRoutes.PUT("/:id", quotations.Update)
		quotationRoutes.PATCH("/:id/status", quotations.UpdateStatus)
		quotationRoutes.DELETE("/:id", quotations.Delete)
		quotationRoutes.GET("/:id/pdf", quotations.PDF)
	}
	userRoutes := protected.Group("/users")
	userRoutes.Use(middleware.RequireAdmin())
	{
		userRoutes.GET("", auth.ListUsers)
		userRoutes.POST("", auth.CreateUser)
		userRoutes.DELETE("/:id", auth.DeleteUser)
	}
	return router, nil
}
