package httpctx

import (
	"bytes"
	"encoding/json"
	"io"
)

func toJson(v interface{}) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}

	buffer, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buffer), nil
}
