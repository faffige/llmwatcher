package bedrock

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func TestSigningTransport_SignsRequests(t *testing.T) {
	// Capture headers sent by the signing transport.
	var gotAuth string
	var gotAmzDate string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAmzDate = r.Header.Get("X-Amz-Date")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"type":"message","model":"claude-sonnet-4-20250514","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer srv.Close()

	creds := credentials.NewStaticCredentialsProvider("AKID", "SECRET", "")
	transport := NewSigningTransport(creds, "us-east-1")
	transport.Inner = http.DefaultTransport

	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	req, err := http.NewRequest("POST", srv.URL+"/model/anthropic.claude-sonnet-4-20250514-v1:0/invoke", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if gotAuth == "" {
		t.Error("expected Authorization header to be set by Sig V4 signer")
	}
	if gotAmzDate == "" {
		t.Error("expected X-Amz-Date header to be set by Sig V4 signer")
	}

	// Verify the Authorization header contains AWS4-HMAC-SHA256.
	if len(gotAuth) < 20 || gotAuth[:16] != "AWS4-HMAC-SHA256" {
		t.Errorf("Authorization header doesn't look like Sig V4: %q", gotAuth)
	}
}

func TestSigningTransport_UsesCredentials(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	creds := credentials.NewStaticCredentialsProvider("TESTKEY", "TESTSECRET", "TESTSESSION")
	transport := NewSigningTransport(creds, "eu-west-1")
	transport.Inner = http.DefaultTransport

	req, _ := http.NewRequest("POST", srv.URL+"/model/test/invoke", nil)
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !contains(gotAuth, "TESTKEY") {
		t.Errorf("expected Authorization to reference access key TESTKEY, got: %q", gotAuth)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSigningTransport_BodyPreserved(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	creds := credentials.NewStaticCredentialsProvider("AKID", "SECRET", "")
	transport := NewSigningTransport(creds, "us-east-1")
	transport.Inner = http.DefaultTransport

	body := []byte(`{"messages":[{"role":"user","content":"test body preservation"}]}`)
	req, _ := http.NewRequest("POST", srv.URL+"/model/test/invoke", bytes.NewReader(body))
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !bytes.Equal(gotBody, body) {
		t.Errorf("body was not preserved after signing\ngot:  %s\nwant: %s", gotBody, body)
	}
}

// Verify the interface is satisfied at compile time.
var _ aws.CredentialsProvider = credentials.NewStaticCredentialsProvider("", "", "")
