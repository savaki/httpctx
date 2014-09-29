package httpctx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type transporter interface {
	RoundTrip(*http.Request) (*http.Response, error)
	CancelRequest(*http.Request)
}

type AuthFunc func(*http.Request) *http.Request

type ErrorMessage struct {
	StatusCode int
	Data       []byte
}

func (e ErrorMessage) Unmarshal(v interface{}) error {
	return json.NewDecoder(bytes.NewReader(e.Data)).Decode(v)
}

func (e ErrorMessage) Error() string {
	return fmt.Sprintf("returned status code => %d", e.StatusCode)
}
