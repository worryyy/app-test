package file

import (
	"mime/multipart"
	"net/textproto"
	"testing"
)

func TestNormalizeContentType(t *testing.T) {
	header := &multipart.FileHeader{Header: textproto.MIMEHeader{}}
	header.Header.Set("Content-Type", "application/octet-stream")
	contentType, err := normalizeContentType(header)
	if err != nil {
		t.Fatalf("normalizeContentType returned error: %v", err)
	}
	if contentType != "image/png" {
		t.Fatalf("unexpected content type: %s", contentType)
	}

	header.Header.Set("Content-Type", "image/gif")
	if _, err := normalizeContentType(header); err == nil {
		t.Fatalf("expected unsupported content type error")
	}
}
