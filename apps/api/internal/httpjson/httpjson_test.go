package httpjson

import (
	"bytes"
	"compress/gzip"
	stderrors "errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
)

type testPayload struct {
	Message string `json:"message"`
}

func gzipBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(raw); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func errorCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) {
		t.Fatalf("error %v is not an app error", err)
	}
	return appErr.Code
}

func TestDecodeGzipJSON(t *testing.T) {
	validJSON := []byte(`{"message":"hello"}`)
	bigJSON := []byte(`{"message":"` + strings.Repeat("a", 4096) + `"}`)

	cases := []struct {
		name     string
		body     []byte
		maxBytes int64
		wantCode string
		wantMsg  string
	}{
		{"valid gzip decodes", gzipBytes(t, validJSON), 32 << 20, "", "hello"},
		{"exactly at cap decodes", gzipBytes(t, validJSON), int64(len(validJSON)), "", "hello"},
		{"invalid gzip rejected", []byte("definitely not gzip"), 32 << 20, "invalid_argument", ""},
		{"truncated gzip rejected", gzipBytes(t, validJSON)[:8], 32 << 20, "invalid_argument", ""},
		{"oversized decompressed rejected", gzipBytes(t, bigJSON), 1024, "resource_exhausted", ""},
		{"invalid JSON inside gzip rejected", gzipBytes(t, []byte("not json")), 32 << 20, "invalid_argument", ""},
		{"trailing data inside gzip rejected", gzipBytes(t, []byte(`{"message":"a"}{"message":"b"}`)), 32 << 20, "invalid_argument", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/ingest", bytes.NewReader(tc.body))
			recorder := httptest.NewRecorder()

			var dst testPayload
			err := DecodeGzipJSON(recorder, request, &dst, tc.maxBytes)
			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("DecodeGzipJSON = %v, want nil", err)
				}
				if dst.Message != tc.wantMsg {
					t.Fatalf("decoded message %q, want %q", dst.Message, tc.wantMsg)
				}
				return
			}
			if err == nil {
				t.Fatal("DecodeGzipJSON = nil, want error")
			}
			if code := errorCode(t, err); code != tc.wantCode {
				t.Fatalf("error code %q, want %q", code, tc.wantCode)
			}
		})
	}
}

func TestDecodeGzipJSONZipBomb(t *testing.T) {
	bomb := gzipBytes(t, []byte(`{"message":"`+strings.Repeat("a", 8<<20)+`"}`))
	if len(bomb) >= 1<<20 {
		t.Fatalf("bomb should compress well, got %d bytes", len(bomb))
	}

	request := httptest.NewRequest("POST", "/ingest", bytes.NewReader(bomb))
	var dst testPayload
	err := DecodeGzipJSON(httptest.NewRecorder(), request, &dst, 1<<20)
	if err == nil {
		t.Fatal("zip bomb accepted")
	}
	if code := errorCode(t, err); code != "resource_exhausted" {
		t.Fatalf("error code %q, want resource_exhausted", code)
	}
}
