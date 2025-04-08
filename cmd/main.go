package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/driftreport/providers"
	"github.com/driftreport/services"
	"github.com/driftreport/utils"
	"github.com/joho/godotenv"
)

func main() {
	//initialize zap logging
	logger := utils.InitZapLog()
	defer logger.Sync() // Flush any buffered log messages

	//initialize AWS EC2 provider
	awsProvider, err := providers.NewAWSProvider()
	if err != nil {
		utils.Logger.Sugar().Errorf("error creating AWS provider: %v", err)
		return
	}

	//initialize drift report service
	svc := services.NewDriftReport(awsProvider)
	err = godotenv.Load(".env")
	if err != nil {
		utils.Logger.Sugar().Errorf("error loading .env file: %v", err)
		return
	}

	if os.Getenv("ENVIRONMENT") == "" {
		utils.Logger.Sugar().Error("environment not set")
		return
	}

	log.Printf("ENVIRONMENT=[%v]", os.Getenv("ENVIRONMENT"))

	//context.WithTimeout() to allow early exit when deadline is exceeded
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = svc.PrintDriftReport(ctx)
	if err != nil {
		utils.Logger.Sugar().Errorf("error printing drift report: %v", err)
		return
	}
}
