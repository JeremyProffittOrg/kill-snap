package main

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func zipDownloadAvailability(head func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error), bucket, key string) (int, string) {
	_, err := head(&s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err == nil {
		return 0, ""
	}
	var apiErr awserr.Error
	if errors.As(err, &apiErr) && (apiErr.Code() == "NotFound" || apiErr.Code() == "NoSuchKey" || apiErr.Code() == "404") {
		return 404, "Zip file has expired or is missing. Generate the project zip again."
	}
	return 502, "Unable to check zip availability. Please retry."
}
