package aws

import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/lttkgp/R2-D2/internal/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

// NewAwsSession initializes an AWS session using env variable and default values as fallback
func NewAwsSession() (*session.Session, error) {
	// Sensible defaults useful for local development
	awsAccessKey := config.GetEnv("AWS_ACCESS_KEY_ID", "DEFAULT_KEY")
	awsSecretKey := config.GetEnv("AWS_SECRET_ACCESS_KEY", "DEFAULT_SECRET")
	awsDefaultRegion := config.GetEnv("AWS_REGION", "ap-south-1")

	awsSession, err := session.NewSession(
		&aws.Config{
			Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
			Region:      aws.String(awsDefaultRegion),
		},
	)

	if err != nil {
		return nil, err
	}

	return awsSession, nil
}
