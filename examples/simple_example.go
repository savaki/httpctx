package main

import (
	"fmt"
	"net/url"

	"code.google.com/p/go.net/context"
	"github.com/savaki/httpctx"
)

func main() {
	weather := struct {
		Values map[string]float32 `json:"main"`
	}{}

	params := url.Values{}
	params.Set("q", "London,uk")

	client := httpctx.NewClient()
	client.Get(context.Background(), "http://api.openweathermap.org/data/2.5/weather", &params, &weather)

	fmt.Printf("The humidity in London is %.1f\n", weather.Values["humidity"])
}
