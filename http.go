package httpctx

import (
	"bytes"
	"code.google.com/p/go.net/context"
	"encoding/json"
	"fmt"
	"github.com/savaki/stormpath-go/auth"
	"io"
	"net/http"
	"net/url"
)

// handles creation of http.Transport instances; provides simple hook that can be overridden for testing
var newTransporter func() transporter = makeTransporterFunc

func makeTransporterFunc() transporter {
	return &http.Transport{}
}

type HttpClient interface {
	Get(ctx context.Context, path string, params *url.Values, v interface{}) error
	Post(ctx context.Context, path string, body interface{}, v interface{}) error
	Put(ctx context.Context, path string, body interface{}, v interface{}) error
	Delete(ctx context.Context, path string) error
}

func NewClient(authFunc auth.AuthFunc) HttpClient {
	return &client{
		authFunc:  authFunc,
		UserAgent: "httpctx-go:0.1",
	}
}

type client struct {
	authFunc  auth.AuthFunc
	UserAgent string
}

func (h *client) Get(ctx context.Context, path string, params *url.Values, v interface{}) error {
	u, err := url.Parse(path)
	if err != nil {
		return err
	}

	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, _ := http.NewRequest("GET", u.String(), nil)
	return h.Do(ctx, req, v)
}

func (h *client) Post(ctx context.Context, path string, data interface{}, v interface{}) error {
	body, err := toJson(data)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", "application/json")
	return h.Do(ctx, req, v)
}

func (h *client) Put(ctx context.Context, path string, data interface{}, v interface{}) error {
	body, err := toJson(data)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("PUT", path, body)
	req.Header.Set("Content-Type", "application/json")
	return h.Do(ctx, req, v)
}

func (h *client) Delete(ctx context.Context, path string) error {
	req, _ := http.NewRequest("DELETE", path, nil)
	return h.Do(ctx, req, nil)
}

type response struct {
	resp *http.Response
	err  error
}

func (h *client) Do(ctx context.Context, req *http.Request, v interface{}) error {
	if req.URL.String() == "" {
		return fmt.Errorf("invalid attempt to %s to an empty url", req.Method)
	}

	req.Header.Set("User-Agent", h.UserAgent)
	req.Header.Set("Accept", "application/json")

	if h.authFunc != nil {
		h.authFunc(req)
	}

	// send the request on a new custom transport; result will be pumped to the ch channel
	tr := newTransporter()
	ch := make(chan response, 1)
	defer close(ch)

	go func() {
		resp, err := tr.RoundTrip(req)
		ch <- response{resp: resp, err: err}
	}()

	// wait for either response or the request to be canceled
	var resp *http.Response
	var err error

	select {
	case <-ctx.Done():
		fmt.Println("tr.CancelRequest(req)")
		tr.CancelRequest(req)
		<-ch
		return ctx.Err()
	case r := <-ch:
		fmt.Println("resp = r.resp")
		resp = r.resp
		err = r.err
	}

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// manually follow a 302 redirect
	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		return h.Get(ctx, location, nil, v)
	}

	// process this request
	buf := bytes.NewBuffer([]byte{})
	io.Copy(buf, resp.Body)
	data := buf.Bytes()

	if !ok(resp.StatusCode) {
		errMsg := &ErrorMessage{StatusCode: resp.StatusCode, Data: data}
		err = json.Unmarshal(data, errMsg)
		if err != nil {
			return err
		}
		return errMsg
	}

	if v != nil {
		err = json.Unmarshal(data, v)
	}

	return err
}

func ok(statusCode int) bool {
	firstDigit := statusCode / 100
	return firstDigit == 2
}
