httpctx
=======

[![GoDoc](https://godoc.org/github.com/savaki/httpctx?status.svg)](https://godoc.org/github.com/savaki/httpctx)

Provides a json-aware http client that utilizes Google's excellent ```context``` package, [code.google.com/p/go.net/context](http://code.google.com/p/go.net/context).  There's an excellent [blog entry](http://blog.golang.org/context) on the context package [here](http://blog.golang.org/context).

The intent of this library is to simplify the task of working with json apis with go in a robust manner.  By utilizing the ```context``` package, we inherit the ability to handle cancellation signals and deadlines across api boundaries and go routines.

## Features:

* supports Google's ```context``` library
* natively handles json request/response
* allows for optional user defined authentication function

## Simple Example - Using openweathermap, retrieve the weather in London

```
package main

import (
	"code.google.com/p/go.net/context"
	"fmt"
	"github.com/savaki/httpctx"
	"net/url"
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
```

In this example, we execute a simple json request to get the weather in London.  httpctx nicely handles the json unmarshal for us as part of the get request. 

## Simple Example - POST'ing json content

Assuming the service, http://neato.com/api/foo returns the following body:

```
{
	"status": "ok"
	"message": "yep, works!"}
```

We can use httpctx to post to this api as follows:

```
package main

import (
	"code.google.com/p/go.net/context"
	"fmt"
	"github.com/savaki/httpctx"
	"net/url"
)

func main() {
	body := map[string]string{"hello":"world"}
	result := map[string]string{}

	client := httpctx.NewClient()
	client.Post(context.Background(), "http://some-api-here.com", body, &result)	
	fmt.Println("the status is ", result["status"])}
```

## With Timeouts

The real value is when you start needing to handle coordinations or provide SLAs.  We'll update the previous example to support timeouts.  We definitely want to make sure our service returns within our specified SLA.

```
package main

import (
	"code.google.com/p/go.net/context"
	"fmt"
	"github.com/savaki/httpctx"
	"net/url"
	"time"
)

func main() {
	weather := struct {
		Values map[string]float32 `json:"main"`
	}{}

	params := url.Values{}
	params.Set("q", "London,uk")

	ctx, _ := context.WithTimeout(context.Background(), 3 * time.Second)
	client := httpctx.NewClient()
	err := client.Get(ctx, "http://api.openweathermap.org/data/2.5/weather", &params, &weather)
	
	if err == nil {
		fmt.Printf("The humidity in London is %.1f\n", weather.Values["humidity"])
	}
}
```

Getting better.  Here, with the addition of a few lines of code, we can now control how long we'll spend waiting for our request.  Once timeout has been reached, the outstanding connection will be closed and an error returned.

## With Cancelation

Let's take even a more complicated example, let's suppose that our timeout is now part of a group of go-routines collectively responding to a user's query.  When the user cancel's their request, we want to propagate that cancelation to all the go-routines to not consume any more resources than we need to.  Here's how we might do that:

```
package main

import (
	"code.google.com/p/go.net/context"
	"fmt"
	"github.com/savaki/httpctx"
	"net/url"
	"time"
)

func main() {
	weather := struct {
		Values map[string]float32 `json:"main"`
	}{}

	params := url.Values{}
	params.Set("q", "London,uk")

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	client := httpctx.NewClient()
	go func() { cancel() }() // cancel the request
	err := client.Get(ctx, "http://api.openweathermap.org/data/2.5/weather", &params, &weather)
	
	if err != nil {
		fmt.Println("The call was cancel at the user's request")
	}
}
```

