package entities

type AppConfig struct {
	Environment string `env:"ENVIRONMENT"`
	AWSRegion   string `env:"AWS_REGION"`
}
