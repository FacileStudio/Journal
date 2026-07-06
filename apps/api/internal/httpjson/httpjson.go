package httpjson

import (
	"compress/gzip"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
)

const maxJSONBodyBytes int64 = 8 << 20

var errDecompressedTooLarge = stderrors.New("decompressed body too large")

func DecodeJSON(w http.ResponseWriter, request *http.Request, dst any) error {
	defer request.Body.Close()
	request.Body = http.MaxBytesReader(w, request.Body, maxJSONBodyBytes)
	return decodeStrict(request.Body, dst)
}

func DecodeGzipJSON(w http.ResponseWriter, request *http.Request, dst any, maxDecompressedBytes int64) error {
	defer request.Body.Close()
	request.Body = http.MaxBytesReader(w, request.Body, maxJSONBodyBytes)

	decompressed, err := gzip.NewReader(request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return errors.TooLarge("request body too large")
		}
		return errors.Invalid("invalid gzip body")
	}
	defer decompressed.Close()

	return decodeStrict(&cappedReader{reader: decompressed, remaining: maxDecompressedBytes}, dst)
}

func decodeStrict(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return mapDecodeError(err, "invalid JSON body")
	}
	if err := decoder.Decode(new(struct{})); !stderrors.Is(err, io.EOF) {
		return mapDecodeError(err, "request body must contain a single JSON object")
	}
	return nil
}

func mapDecodeError(err error, invalidMessage string) error {
	var maxBytesErr *http.MaxBytesError
	if stderrors.As(err, &maxBytesErr) {
		return errors.TooLarge("request body too large")
	}
	if stderrors.Is(err, errDecompressedTooLarge) {
		return errors.TooLarge("decompressed body too large")
	}
	if stderrors.Is(err, gzip.ErrHeader) || stderrors.Is(err, gzip.ErrChecksum) {
		return errors.Invalid("invalid gzip body")
	}
	return errors.Invalid(invalidMessage)
}

type cappedReader struct {
	reader    io.Reader
	remaining int64
}

func (c *cappedReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.remaining -= int64(n)
	if c.remaining < 0 {
		return n, errDecompressedTooLarge
	}
	return n, err
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) {
		appErr = errors.Internal("internal server error", err)
	}

	WriteJSON(w, errors.Status(appErr), map[string]any{
		"error": map[string]string{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	})
}
