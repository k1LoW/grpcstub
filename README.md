# grpcstub [![Go Reference](https://pkg.go.dev/badge/github.com/k1LoW/grpcstub.svg)](https://pkg.go.dev/github.com/k1LoW/grpcstub)

grpcstub provides gRPC server and client conn ( `*grpc.ClientConn` ) for stubbing, for testing in Go.

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
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

## Test data

- https://github.com/grpc/grpc-go/blob/master/examples/route_guide/routeguide/route_guide.proto

## References

- [monlabs/grpc-mock](https://github.com/monlabs/grpc-mock): Run a gRPC mock server by using protobuf reflection.
