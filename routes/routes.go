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
	templates *controllers.QuotationTemplateController,
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
	protected.GET("/dashboard", middleware.RequirePermission(middleware.ResourceDashboard, middleware.ActionRead), dashboard.Get)
	templateRoutes := protected.Group("/templates", middleware.RequirePermission(middleware.ResourceTemplates, middleware.ActionRead))
	{
		templateRoutes.GET("", templates.List)
		templateRoutes.GET("/:id", templates.Get)
	}
	protected.GET("/template-logos/:filename", templates.LogoAsset)
	adminTemplateRoutes := protected.Group("/templates", middleware.RequirePermission(middleware.ResourceTemplates, middleware.ActionWrite))
	{
		adminTemplateRoutes.POST("", templates.Create)
		adminTemplateRoutes.PUT("/:id", templates.Update)
		adminTemplateRoutes.POST("/:id/logo", templates.UploadLogo)
		adminTemplateRoutes.PATCH("/:id/active", templates.SetActive)
		adminTemplateRoutes.PATCH("/:id/default", templates.SetDefault)
		adminTemplateRoutes.DELETE("/:id", templates.Delete)
	}
	customerRoutes := protected.Group("/customers", middleware.RequirePermission(middleware.ResourceCustomers, middleware.ActionRead))
	{
		customerRoutes.GET("", customers.List)
		customerRoutes.GET("/:id", customers.Get)
	}
	customerRoutes.POST("", middleware.RequirePermission(middleware.ResourceCustomers, middleware.ActionWrite), customers.Create)
	customerRoutes.PUT("/:id", middleware.RequirePermission(middleware.ResourceCustomers, middleware.ActionWrite), customers.Update)
	customerRoutes.DELETE("/:id", middleware.RequirePermission(middleware.ResourceCustomers, middleware.ActionDelete), customers.Delete)
	quotationRoutes := protected.Group("/quotations", middleware.RequirePermission(middleware.ResourceQuotations, middleware.ActionRead))
	{
		quotationRoutes.GET("", quotations.List)
		quotationRoutes.GET("/:id", quotations.Get)
		quotationRoutes.GET("/:id/pdf", quotations.PDF)
	}
	quotationRoutes.POST("", middleware.RequirePermission(middleware.ResourceQuotations, middleware.ActionWrite), quotations.Create)
	quotationRoutes.PUT("/:id", middleware.RequirePermission(middleware.ResourceQuotations, middleware.ActionWrite), quotations.Update)
	quotationRoutes.PATCH("/:id/status", middleware.RequirePermission(middleware.ResourceQuotations, middleware.ActionApprove), quotations.UpdateStatus)
	quotationRoutes.DELETE("/:id", middleware.RequirePermission(middleware.ResourceQuotations, middleware.ActionDelete), quotations.Delete)
	userRoutes := protected.Group("/users", middleware.RequirePermission(middleware.ResourceUsers, middleware.ActionRead))
	{
		userRoutes.GET("", auth.ListUsers)
		userRoutes.POST("", middleware.RequirePermission(middleware.ResourceUsers, middleware.ActionWrite), auth.CreateUser)
		userRoutes.DELETE("/:id", middleware.RequirePermission(middleware.ResourceUsers, middleware.ActionDelete), auth.DeleteUser)
	}
	return router, nil
}
