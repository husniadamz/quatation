package main

import (
	"log"

	"quotation/config"
	"quotation/controllers"
	"quotation/models"
	"quotation/repositories"
	"quotation/routes"
	"quotation/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	db, err := config.ConnectDatabase(cfg)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.Quotation{}, &models.QuotationItem{},
		&models.User{}, &models.AuthSession{}, &models.InstallState{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	customerRepo := repositories.NewCustomerRepository(db)
	quotationRepo := repositories.NewQuotationRepository(db)
	authRepo := repositories.NewAuthRepository(db)
	customerService := services.NewCustomerService(customerRepo)
	quotationService := services.NewQuotationService(quotationRepo, customerRepo)
	authService := services.NewAuthService(authRepo)
	customerController := controllers.NewCustomerController(customerService)
	quotationController := controllers.NewQuotationController(quotationService)
	authController := controllers.NewAuthController(authService)
	dashboardController := controllers.NewDashboardController(quotationService, customerService)

	router, err := routes.New(customerController, quotationController, authController, dashboardController, authService)
	if err != nil {
		log.Fatalf("configure router: %v", err)
	}
	log.Printf("quotation API listening on %s", cfg.Address)
	if err := router.Run(cfg.Address); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
