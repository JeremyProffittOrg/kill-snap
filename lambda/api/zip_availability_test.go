package main

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"testing"
)

func TestZipAvailability(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"fresh", nil, 0}, {"expired", awserr.New("NotFound", "", nil), 404},
		{"denied", awserr.New("AccessDenied", "", nil), 502}, {"transient", errors.New("timeout"), 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			head := func(in *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				if aws.StringValue(in.Key) != "projects/legacy.zip" {
					t.Fatal("legacy key changed")
				}
				return &s3.HeadObjectOutput{}, tc.err
			}
			status, _ := zipDownloadAvailability(head, "kill-snap", "projects/legacy.zip")
			if status != tc.status {
				t.Fatalf("status=%d", status)
			}
		})
	}
}
