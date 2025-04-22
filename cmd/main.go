package main

import (
	"context"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/driftreport/entities"
	"github.com/driftreport/providers"
	"github.com/driftreport/services"
	"github.com/driftreport/utils"
	"github.com/joho/godotenv"
)

func main() {
	//initialize zap logging
	logger := utils.InitZapLog()
	defer logger.Sync() // Flush any buffered log messages

	// Load environment variables from .env file
	err := godotenv.Load(".env")
	if err != nil {
		utils.Logger.Sugar().Errorf("error loading .env file: %v", err)
		return
	}

	var appConfig entities.AppConfig
	if err := env.Parse(&appConfig); err != nil {
		utils.Logger.Sugar().Errorf("error reading the environment variables: %v", err)
		return
	}

	if appConfig.Environment == "" {
		utils.Logger.Sugar().Error("environment not set")
		return
	}
	log.Printf("ENVIRONMENT=[%v]", appConfig.Environment)

	//initialize AWS EC2 provider
	awsProvider, err := providers.NewAWSProvider(appConfig.AWSRegion)
	if err != nil {
		utils.Logger.Sugar().Errorf("error creating AWS provider: %v", err)
		return
	}

	//initialize drift report service
	svc := services.NewDriftReportService(awsProvider)

	//context.WithTimeout() to allow early exit when deadline is exceeded
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = svc.PrintDriftReport(ctx)
	if err != nil {
		utils.Logger.Sugar().Errorf("error printing drift report: %v", err)
		return
	}
}
