package svc

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/distr-sh/distr/internal/env"
	. "github.com/onsi/gomega"
)

// A presign client inherits the API options of the client it was built from, but its finalize step
// has no Signing middleware. The accept-encoding workaround must not attach itself to one there,
// or presigning fails and every redirected blob download returns a 500.
func TestResignForGCPAllowsPresigning(t *testing.T) {
	g := NewWithT(t)

	client := s3.New(s3.Options{}, s3ClientOptions(env.S3Config{
		Region:          "auto",
		Endpoint:        new("https://storage.googleapis.com"),
		AccessKeyID:     new("GOOG1ETEST"),
		SecretAccessKey: new("test-secret"),
		UsePathStyle:    true,
	}), resignForGCP)

	request, err := s3.NewPresignClient(client).PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: new("test-bucket"),
		Key:    new("sha256:0000000000000000000000000000000000000000000000000000000000000000"),
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(request.URL).To(ContainSubstring("X-Amz-Signature="))
}
