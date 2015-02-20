package httpctx

import (
	"net/url"

	"golang.org/x/net/context"
)

// added Mock version of httpctx to simplify testing
type Mock struct {
	Err     error
	Payload interface{}
	Body    string
}

func (m Mock) Get(ctx context.Context, path string, params *url.Values, v interface{}) error {
	return m.Do(ctx, "GET", path, params, nil, v)
}

func (m Mock) Post(ctx context.Context, path string, payload interface{}, v interface{}) error {
	return m.Do(ctx, "POST", path, nil, payload, v)
}

func (m Mock) Put(ctx context.Context, path string, payload interface{}, v interface{}) error {
	return m.Do(ctx, "PUT", path, nil, payload, v)
}

func (m Mock) Delete(ctx context.Context, path string) error {
	return m.Do(ctx, "DELETE", path, nil, nil, nil)
}

func (m Mock) Do(ctx context.Context, method, path string, params *url.Values, payload interface{}, v interface{}) error {
	if m.Body != "" {
		err := marshal([]byte(m.Body), v)
		if err != nil {
			return err
		}
	}

	return m.Err
}
