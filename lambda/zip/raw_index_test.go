package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"testing"
)

func TestRawDirectoryReuse(t *testing.T) {
	calls := 0
	list := func(in *s3.ListObjectsV2Input, next func(*s3.ListObjectsV2Output, bool) bool) error {
		calls++
		prefix := aws.StringValue(in.Prefix)
		for page := 0; page < 2; page++ {
			var objects []*s3.Object
			for i := page * 250; i < (page+1)*250; i++ {
				objects = append(objects, &s3.Object{Key: aws.String(fmt.Sprintf("%sIMG%d.CR3", prefix, i))})
			}
			next(&s3.ListObjectsV2Output{Contents: objects}, page == 1)
		}
		return nil
	}
	index := newRawDirectoryIndex(list)
	for i := 0; i < 500; i++ {
		got, err := index.find(fmt.Sprintf("projects/a/img%d.jpg", i))
		if err != nil || len(got) != 1 {
			t.Fatalf("image %d: %v %v", i, got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("LIST sequences=%d", calls)
	}
	if _, err := index.find("projects/b/img1.jpg"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("second directory did not list independently")
	}
	if _, err := newRawDirectoryIndex(list).find("projects/a/img1.jpg"); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatal("fresh export reused an old cache")
	}
}

func TestRawListFailureIsNotCached(t *testing.T) {
	calls := 0
	index := newRawDirectoryIndex(func(_ *s3.ListObjectsV2Input, _ func(*s3.ListObjectsV2Output, bool) bool) error {
		calls++
		return errors.New("denied")
	})
	for i := 0; i < 2; i++ {
		if _, err := index.find("projects/a/img.jpg"); err == nil {
			t.Fatal("list failure swallowed")
		}
	}
	if calls != 2 {
		t.Fatal("failed listing cached")
	}
}

func TestProjectZipKeys(t *testing.T) {
	if got := projectZipKey("a", "name", "2026-10-13", 1, 1); got != "project-zips/a/name_2026-10-13.zip" {
		t.Fatal(got)
	}
	if got := projectZipKey("a", "name", "2026-10-13", 2, 3); got != "project-zips/a/name_2026-10-13_part2.zip" {
		t.Fatal(got)
	}
}
