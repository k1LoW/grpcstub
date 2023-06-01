# grpcstub [![Go Reference](https://pkg.go.dev/badge/github.com/k1LoW/grpcstub.svg)](https://pkg.go.dev/github.com/k1LoW/grpcstub) ![Coverage](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/grpcstub/coverage.svg) ![Code to Test Ratio](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/grpcstub/ratio.svg) ![Test Execution Time](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/grpcstub/time.svg)

grpcstub provides gRPC server and client conn ( `*grpc.ClientConn` ) for stubbing, for testing in Go.

There is an HTTP version stubbing tool with the same design concept, [httpstub](https://github.com/k1LoW/httpstub).

## Usage

``` go
package myapp_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/k1LoW/grpcstub"
	"github.com/k1LoW/myapp/proto/gen/go/myapp"
)

func TestClient(t *testing.T) {
	ctx := context.Background()
	ts := grpcstub.NewServer(t, "path/to/*.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello", "location": map[string]interface{}{"latitude": 10, "longitude": 13}})

	client := myapp.NewRouteGuideClient(ts.Conn())
	if _, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	}); err != nil {
		t.Fatal(err)
	}
	{
		got := len(ts.Requests())
		if want := 1; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	req := ts.Requests()[0]
	{
		got := int32(req.Message["longitude"].(float64))
		if want := int32(13); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}
```

## Dynamic Response

grpcstub can return responses dynamically using the protocol buffer schema.

### Dynamic response to all requests

``` go
ts := grpcstub.NewServer(t, "path/to/*.proto")
t.Cleanup(func() {
	ts.Close()
})
ts.ResponseDynamic()
```

### Dynamic response to a request to a specific method (rpc)

``` go
ts := grpcstub.NewServer(t, "path/to/*.proto")
t.Cleanup(func() {
	ts.Close()
})
ts.Method("GetFeature").ResponseDynamic()
```

### Dynamic response with your own generators

``` go
ts := grpcstub.NewServer(t, "path/to/*.proto")
t.Cleanup(func() {
	ts.Close()
})
fk := faker.New()
want := time.Now()
opts := []GeneratorOption{
	Generator("*_id", func(r *Request) interface{} {
		return fk.UUID().V4()
	}),
	Generator("*_time", func(r *Request) interface{} {
		return want
	}),
}
ts.ResponseDynamic(opts...)
```

## Test data

- https://github.com/grpc/grpc-go/blob/master/examples/route_guide/routeguide/route_guide.proto

## References

- [monlabs/grpc-mock](https://github.com/monlabs/grpc-mock): Run a gRPC mock server by using protobuf reflection.
