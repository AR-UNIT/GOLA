package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Custom Endpoint Resolver for MinIO
type MinioResolver struct {
	EndpointURL string
}

func (m MinioResolver) ResolveEndpoint(service, region string) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL:           m.EndpointURL,
		SigningRegion: region,
	}, nil
}

func main() {
	// Load configuration with custom endpoint resolver for MinIO.
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"), // Dummy region for MinIO
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
		config.WithEndpointResolver(MinioResolver{EndpointURL: "http://127.0.0.1:9000"}),
	)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create the S3 client and enable path style addressing
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Open the image file for upload
	//file, err := os.Open("myimage.jpg")
	file, err := os.Open("scripts/myimage.jpg")
	if err != nil {
		log.Fatal("Failed to open file:", err)
	}
	defer file.Close()

	// Upload file to the "images" bucket in MinIO
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String("images"),
		Key:    aws.String("myimage.jpg"),
		Body:   file,
	})
	if err != nil {
		log.Fatal("Upload failed:", err)
	}

	fmt.Println("Upload successful!")
}
