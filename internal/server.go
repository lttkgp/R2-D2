package main

import (
	"log"
	"os"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/lttkgp/R2-D2/internal/facebook"
	"github.com/lttkgp/R2-D2/pkg/swagger/server/restapi"
	"github.com/lttkgp/R2-D2/pkg/swagger/server/restapi/operations"
	"github.com/robfig/cron/v3"
)

func main() {
	cronLogger := cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger)))
	_, err := c.AddFunc("@every 10h", facebook.BootstrapDb)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = c.AddFunc("@every 10s", facebook.DispatchFreshPosts)
	if err != nil {
		log.Fatalln(err)
	}
	c.Start()

	// Initialize Swagger
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	api := operations.NewHelloAPI(swaggerSpec)
	server := restapi.NewServer(api)

	defer func() {
		if err := server.Shutdown(); err != nil {
			// error handle
			log.Fatalln(err)
		}
	}()

	server.Port = 8080
	api.CheckHealthHandler = operations.CheckHealthHandlerFunc(Health)

	// Start server which listening
	if err := server.Serve(); err != nil {
		log.Fatalln(err)
	}
}

//Health route returns OK
func Health(operations.CheckHealthParams) middleware.Responder {
	return operations.NewCheckHealthOK().WithPayload("OK")
}
