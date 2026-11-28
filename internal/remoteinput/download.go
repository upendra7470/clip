// Package remoteinput securely materializes a remote document for the local
// extraction pipeline.
package remoteinput

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/upendra7470/clip/internal/security"
)

const maxRedirects = 5

type Download struct {
	Path    string
	Source  string
	cleanup func()
}

func (download Download) Cleanup() {
	if download.cleanup != nil {
		download.cleanup()
	}
}

// Fetch downloads one public HTTPS document into a private temporary
// directory. DNS and every redirect target are checked to prevent access to
// loopback, private, link-local, and other non-public networks.
func Fetch(ctx context.Context, rawURL string) (Download, error) {
	parsed, err := validateURL(rawURL)
	if err != nil {
		return Download{}, err
	}

	dialer := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:               nil,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: 15 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("validate remote address: %w", err)
			}
			addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolve remote host: %w", err)
			}
			for _, address := range addresses {
				if !publicIP(address) {
					return nil, fmt.Errorf("remote host resolves to a non-public address")
				}
			}
			if len(addresses) == 0 {
				return nil, fmt.Errorf("remote host has no IP address")
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Minute,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("remote input exceeded %d redirects", maxRedirects)
			}
			_, err := validateURL(request.URL.String())
			return err
		},
	}
	defer transport.CloseIdleConnections()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Download{}, fmt.Errorf("create remote request: %w", err)
	}
	request.Header.Set("User-Agent", "clip-document-extractor")
	response, err := client.Do(request)
	if err != nil {
		return Download{}, fmt.Errorf("download remote input: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Download{}, fmt.Errorf("remote input returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > security.MaxInputFileSize {
		return Download{}, fmt.Errorf("remote input exceeds %d bytes", security.MaxInputFileSize)
	}

	extension, err := documentExtension(response.Request.URL, response.Header.Get("Content-Type"))
	if err != nil {
		return Download{}, err
	}
	directory, err := os.MkdirTemp("", "clip-remote-*")
	if err != nil {
		return Download{}, fmt.Errorf("create private remote workspace: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	root, err := os.OpenRoot(directory)
	if err != nil {
		cleanup()
		return Download{}, fmt.Errorf("open private remote workspace: %w", err)
	}
	name := "document" + extension
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	rootCloseErr := root.Close()
	if err != nil {
		cleanup()
		return Download{}, fmt.Errorf("create remote document: %w", err)
	}
	if rootCloseErr != nil {
		_ = file.Close()
		cleanup()
		return Download{}, fmt.Errorf("close private remote workspace: %w", rootCloseErr)
	}
	path := filepath.Join(directory, name)
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, security.MaxInputFileSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		cleanup()
		return Download{}, fmt.Errorf("save remote input: %w", copyErr)
	}
	if closeErr != nil {
		cleanup()
		return Download{}, fmt.Errorf("close remote input: %w", closeErr)
	}
	if written > security.MaxInputFileSize {
		cleanup()
		return Download{}, fmt.Errorf("remote input exceeds %d bytes", security.MaxInputFileSize)
	}
	if written == 0 {
		cleanup()
		return Download{}, fmt.Errorf("remote input is empty")
	}
	return Download{Path: path, Source: response.Request.URL.String(), cleanup: cleanup}, nil
}

func validateURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid remote input URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("remote input requires an https URL")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("remote input URL must not contain credentials")
	}
	return parsed, nil
}

func publicIP(address net.IP) bool {
	return address != nil && !address.IsLoopback() && !address.IsPrivate() &&
		!address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast() && !address.IsUnspecified()
}

func documentExtension(location *url.URL, contentType string) (string, error) {
	extension := strings.ToLower(filepath.Ext(location.Path))
	if supportedExtension(extension) {
		return extension, nil
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	switch mediaType {
	case "application/pdf":
		return ".pdf", nil
	case "text/plain":
		return ".txt", nil
	case "text/markdown":
		return ".md", nil
	case "text/html":
		return ".html", nil
	case "application/json":
		return ".json", nil
	case "application/xml", "text/xml":
		return ".xml", nil
	case "text/csv":
		return ".csv", nil
	case "image/png":
		return ".png", nil
	case "image/jpeg":
		return ".jpg", nil
	case "image/gif":
		return ".gif", nil
	case "image/tiff":
		return ".tiff", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", fmt.Errorf("remote input has no supported file extension or content type")
	}
}

func supportedExtension(extension string) bool {
	switch extension {
	case ".pdf", ".docx", ".ppt", ".pptx", ".odt", ".rtf", ".txt", ".md", ".markdown", ".csv", ".xlsx", ".ods", ".json", ".xml", ".html", ".htm", ".yaml", ".yml", ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff", ".webp":
		return true
	default:
		return false
	}
}
