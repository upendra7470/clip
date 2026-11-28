package remoteinput

import (
	"net"
	"net/url"
	"testing"
)

func TestValidateURL(t *testing.T) {
	if _, err := validateURL("https://example.com/report.pdf"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"http://example.com/report.pdf",
		"https://user:password@example.com/report.pdf",
		"not a URL",
	} {
		if _, err := validateURL(value); err == nil {
			t.Fatalf("validateURL(%q) succeeded", value)
		}
	}
}

func TestPublicIP(t *testing.T) {
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "169.254.1.1", "::1", "fc00::1"} {
		if publicIP(net.ParseIP(value)) {
			t.Fatalf("publicIP(%s) = true", value)
		}
	}
	if !publicIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public address was rejected")
	}
}

func TestDocumentExtension(t *testing.T) {
	location, _ := url.Parse("https://example.com/download/report.PDF?token=x")
	extension, err := documentExtension(location, "application/octet-stream")
	if err != nil || extension != ".pdf" {
		t.Fatalf("extension = %q, err = %v", extension, err)
	}
	location, _ = url.Parse("https://example.com/download")
	extension, err = documentExtension(location, "image/png")
	if err != nil || extension != ".png" {
		t.Fatalf("extension = %q, err = %v", extension, err)
	}
}
