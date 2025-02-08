package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)



func (cfg *apiConfig)getDefaultS3Client() *s3.Client{
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("error creating default S3 config")
	}
	awsConfig.Region = cfg.s3Region

	s3Client  := s3.NewFromConfig(awsConfig)

	return s3Client
}