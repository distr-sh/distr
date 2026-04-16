package svc

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/distr-sh/distr/internal/env"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

func newS3Client(ctx context.Context) *s3.Client {
	s3Config := env.RegistryS3Config()

	opts := []func(*s3.Options){s3ClientOptions(s3Config)}
	if s3Config.ResignForGCP {
		opts = append(opts, resignForGCP)
	}

	if config, err := awsconfig.LoadDefaultConfig(ctx); err != nil {
		return s3.New(s3.Options{}, opts...)
	} else {
		otelaws.AppendMiddlewares(&config.APIOptions)
		return s3.NewFromConfig(config, opts...)
	}
}

func s3ClientOptions(s3Config env.S3Config) func(o *s3.Options) {
	return func(o *s3.Options) {
		o.Region = s3Config.Region
		o.BaseEndpoint = s3Config.Endpoint
		o.UsePathStyle = s3Config.UsePathStyle
		if s3Config.RequestChecksumCalculationWhenRequired {
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		}
		if s3Config.ResponseChecksumValidationWhenRequired {
			o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		}
		if s3Config.AccessKeyID != nil && s3Config.SecretAccessKey != nil {
			o.Credentials = aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(*s3Config.AccessKeyID, *s3Config.SecretAccessKey, ""),
			)
		}
	}
}
