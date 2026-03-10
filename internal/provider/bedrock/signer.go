package bedrock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const bedrockService = "bedrock"

// SigningTransport is an http.RoundTripper that applies AWS Sig V4 signing
// to every outgoing request. It wraps an inner transport (typically http.DefaultTransport).
type SigningTransport struct {
	Inner       http.RoundTripper
	Credentials aws.CredentialsProvider
	Region      string
	Signer      *v4.Signer
}

// NewSigningTransport creates a transport that signs requests with AWS Sig V4.
func NewSigningTransport(creds aws.CredentialsProvider, region string) *SigningTransport {
	return &SigningTransport{
		Inner:       http.DefaultTransport,
		Credentials: creds,
		Region:      region,
		Signer:      v4.NewSigner(),
	}
}

func (t *SigningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read and buffer the body so we can hash it for signing.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body for signing: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	hash := sha256Hash(bodyBytes)

	creds, err := t.Credentials.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("retrieving AWS credentials: %w", err)
	}

	err = t.Signer.SignHTTP(context.Background(), creds, req, hash, bedrockService, t.Region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("signing request: %w", err)
	}

	return t.Inner.RoundTrip(req)
}

func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
