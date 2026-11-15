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
	if err := db.AutoMigrate(&models.Customer{}, &models.Quotation{}, &models.QuotationItem{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	customerRepo := repositories.NewCustomerRepository(db)
	quotationRepo := repositories.NewQuotationRepository(db)
	customerService := services.NewCustomerService(customerRepo)
	quotationService := services.NewQuotationService(quotationRepo, customerRepo)
	customerController := controllers.NewCustomerController(customerService)
	quotationController := controllers.NewQuotationController(quotationService)

	router, err := routes.New(customerController, quotationController)
	if err != nil {
		log.Fatalf("configure router: %v", err)
	}
	log.Printf("quotation API listening on %s", cfg.Address)
	if err := router.Run(cfg.Address); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
