package main

import (
	"context"
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
	if err := db.AutoMigrate(&models.Customer{}, &models.Quotation{}, &models.QuotationItem{}, &models.Company{},
		&models.User{}, &models.AuthSession{}, &models.InstallState{}, &models.QuotationTemplate{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	customerRepo := repositories.NewCustomerRepository(db)
	quotationRepo := repositories.NewQuotationRepository(db)
	templateRepo := repositories.NewQuotationTemplateRepository(db)
	if err := templateRepo.EnsureStarterTemplate(context.Background()); err != nil {
		log.Fatalf("ensure starter quotation template: %v", err)
	}
	authRepo := repositories.NewAuthRepository(db)
	companyRepo := repositories.NewCompanyRepository(db)
	companyService := services.NewCompanyService(companyRepo)
	customerService := services.NewCustomerService(customerRepo)
	templateService := services.NewQuotationTemplateService(templateRepo, "data/template-logos")
	quotationService := services.NewQuotationService(quotationRepo, customerRepo, templateRepo)
	authService := services.NewAuthService(authRepo)
	customerController := controllers.NewCustomerController(customerService)
	quotationController := controllers.NewQuotationController(quotationService)
	templateController := controllers.NewQuotationTemplateController(templateService, companyService)
	authController := controllers.NewAuthController(authService)
	companyController := controllers.NewCompanyController(companyService)
	dashboardController := controllers.NewDashboardController(quotationService, customerService)

	router, err := routes.New(customerController, quotationController, authController, dashboardController,
		templateController, authService, companyController)
	if err != nil {
		log.Fatalf("configure router: %v", err)
	}
	log.Printf("quotation API listening on %s", cfg.Address)
	if err := router.Run(cfg.Address); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
