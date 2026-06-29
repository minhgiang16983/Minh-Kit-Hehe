package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// LoadFromAWSSecretsManager loads environment variables from AWS Secrets Manager
func LoadFromAWSSecretsManager(ctx context.Context) error {
	if strings.ToUpper(os.Getenv("ENABLE_AWS_SECRET_MANAGER")) != "TRUE" {
		return nil
	}

	secretARN := os.Getenv("AWS_SECRET_ARN")
	if secretARN == "" {
		return fmt.Errorf("AWS_SECRET_ARN is required")
	}

	var awsCfg aws.Config
	var err error

	region := os.Getenv("AWS_REGION")
	if region != "" {
		awsCfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	} else {
		awsCfg, err = awsConfig.LoadDefaultConfig(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	if awsCfg.Region == "" {
		return fmt.Errorf("AWS_REGION is required (set AWS_REGION environment variable or configure AWS config)")
	}

	client := secretsmanager.NewFromConfig(awsCfg)
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	if result.SecretString == nil {
		return fmt.Errorf("secret value is empty")
	}
	secretString := *result.SecretString

	var secrets map[string]interface{}
	if err := json.Unmarshal([]byte(secretString), &secrets); err != nil {
		return fmt.Errorf("failed to parse secret as JSON: %w", err)
	}

	secretPath := os.Getenv("AWS_SECRET_PATH")
	if secretPath != "" {
		serviceConfig, ok := secrets[secretPath].(map[string]interface{})
		if !ok {
			return fmt.Errorf("secret path '%s' not found or not an object", secretPath)
		}
		secrets = serviceConfig
	}

	for key, value := range secrets {
		var envValue string
		switch v := value.(type) {
		case string:
			envValue = v
		case bool:
			envValue = strconv.FormatBool(v)
		case nil:
			envValue = ""
		default:
			envValue = fmt.Sprintf("%v", v)
		}
		os.Setenv(key, envValue)
	}

	return nil
}
