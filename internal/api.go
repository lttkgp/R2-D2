package main

import (
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/lttkgp/R2-D2/pkg/swagger/server/restapi"
	"github.com/lttkgp/R2-D2/pkg/swagger/server/restapi/operations"
	"go.uber.org/zap"
)

func initializeAPIServer(logger *zap.Logger) {
	// Initialize Swagger
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		logger.Fatal("Failed to parse Swagger config", zap.Error(err))
	}

	api := operations.NewR2d2API(swaggerSpec)
	server := restapi.NewServer(api)

	defer func() {
		if err := server.Shutdown(); err != nil {
			logger.Fatal("Failed to gracefully shut down the server", zap.Error(err))
		}
	}()

	server.Port = 8080
	api.CheckHealthHandler = operations.CheckHealthHandlerFunc(Health)

	// Start server
	if err := server.Serve(); err != nil {
		logger.Fatal("Unable to start the API server", zap.Error(err))
	}
}

//Health route returns OK
func Health(operations.CheckHealthParams) middleware.Responder {
	return operations.NewCheckHealthOK().WithPayload("OK")
}
