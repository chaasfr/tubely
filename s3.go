package main

import (
	"context"
	"log"
	"time"

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

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	 presignedClient := s3.NewPresignClient(s3Client)

	 getObjectInput := s3.GetObjectInput{
	 	Bucket:                     &bucket,
	 	Key:                        &key,
	 }
	 presignedReq, err := presignedClient.PresignGetObject(context.Background(), &getObjectInput, s3.WithPresignExpires(expireTime))
	 if err != nil {
		return "", err
	 }
	 
	 return presignedReq.URL, nil
}