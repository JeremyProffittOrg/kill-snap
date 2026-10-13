package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func projectZipKey(prefix, name, date string, part, count int) string {
	if count == 1 {
		return fmt.Sprintf("project-zips/%s/%s_%s.zip", prefix, name, date)
	}
	return fmt.Sprintf("project-zips/%s/%s_%s_part%d.zip", prefix, name, date, part)
}

type rawDirectoryIndex struct {
	list        func(*s3.ListObjectsV2Input, func(*s3.ListObjectsV2Output, bool) bool) error
	directories map[string]map[string][]string
}

func newRawDirectoryIndex(list func(*s3.ListObjectsV2Input, func(*s3.ListObjectsV2Output, bool) bool) error) *rawDirectoryIndex {
	return &rawDirectoryIndex{list: list, directories: make(map[string]map[string][]string)}
}

func (r *rawDirectoryIndex) find(original string) ([]string, error) {
	if original == "" {
		return nil, nil
	}
	prefix := path.Dir(original) + "/"
	index, ok := r.directories[prefix]
	if !ok {
		index = make(map[string][]string)
		err := r.list(&s3.ListObjectsV2Input{Bucket: aws.String(bucketName), Prefix: aws.String(prefix)}, func(page *s3.ListObjectsV2Output, _ bool) bool {
			for _, object := range page.Contents {
				key := aws.StringValue(object.Key)
				name := path.Base(key)
				if isRAWFile(name) {
					stem := strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))
					index[stem] = append(index[stem], key)
				}
			}
			return true
		})
		if err != nil {
			return nil, err
		}
		r.directories[prefix] = index
	}
	name := path.Base(original)
	stem := strings.ToLower(strings.TrimSuffix(name, path.Ext(name)))
	var matches []string
	for _, key := range index[stem] {
		if key != original {
			matches = append(matches, key)
		}
	}
	return matches, nil
}
