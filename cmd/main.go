package main

import (
	"context"
	"log"
	"os"

	"github.com/driftreport/providers"
	"github.com/driftreport/services"
	"github.com/driftreport/utils"
	"github.com/joho/godotenv"
)

func main() {

	logger := utils.InitZapLog()
	logger.Sync()
	awsProvider, err := providers.NewAWSProvider()
	if err != nil {
		utils.Logger.Sugar().Errorf("error creating AWS provider: %v", err)
		return
	}

	svc := services.NewDriftReport(awsProvider)
	err = godotenv.Load(".env")
	if err != nil {
		utils.Logger.Sugar().Errorf("error loading .env file: %v", err)
		return
	}

	if os.Getenv("ENVIRONMENT") == "" {
		log.Println("environment not set")
		return
	}

	log.Printf("ENVIRONMENT=[%v]", os.Getenv("ENVIRONMENT"))
	err = svc.GenerateDriftReport(context.Background())
	if err != nil {
		utils.Logger.Sugar().Errorf("error running drift report: %v", err)
		return
	}
}
