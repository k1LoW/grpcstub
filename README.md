# grpcstub [![Go Reference](https://pkg.go.dev/badge/github.com/k1LoW/grpcstub.svg)](https://pkg.go.dev/github.com/k1LoW/grpcstub) ![Coverage](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/grpcstub/coverage.svg) ![Code to Test Ratio](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/grpcstub/ratio.svg) ![Test Execution Time](https://raw.githubusercontent.com/k1LoW/octocovs/main/badges/k1LoW/grpcstub/time.svg)

grpcstub provides gRPC server and client conn ( `*grpc.ClientConn` ) for stubbing, for testing in Go.

There is an HTTP version stubbing tool with the same design concept, [httpstub](https://github.com/k1LoW/httpstub).

## Usage

``` go
package myapp

import (
	"io"
	"net/http"
	"testing"

	"github.com/k1LoW/grpcstub"
	"github.com/k1LoW/myapp/routeguide"
)

func TestClient(t *testing.T) {
	ctx := context.Background()
	ts := grpcstub.NewServer(t, []string{}, "path/to/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello", "location": map[string]interface{}{"latitude": 10, "longitude": 13}})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
		got := req.Message.Get("/longitude").(int32)
		if want := int32(13); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}
```

## Test data

- https://github.com/grpc/grpc-go/blob/master/examples/route_guide/routeguide/route_guide.proto

## References

- [monlabs/grpc-mock](https://github.com/monlabs/grpc-mock): Run a gRPC mock server by using protobuf reflection.
