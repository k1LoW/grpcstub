package grpcstub

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k1LoW/grpcstub/testdata/hello"
	"github.com/k1LoW/grpcstub/testdata/routeguide"
)

func TestResponseDynamic(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").ResponseDynamic()
	want := 5
	responses := []*routeguide.Feature{}
	for i := 0; i < want; i++ {
		client := routeguide.NewRouteGuideClient(ts.Conn())
		res, err := client.GetFeature(ctx, &routeguide.Point{
			Latitude:  10,
			Longitude: 13,
		})
		if err != nil {
			t.Fatal(err)
		}
		responses = append(responses, res)
	}
	{
		got := len(ts.Requests())
		if got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	rc := len(responses)
	for i := 0; i < rc; i++ {
		a := responses[i%rc]
		b := responses[(i+1)%rc]
		opts := []cmp.Option{
			cmpopts.IgnoreFields(routeguide.Feature{}, "state", "sizeCache", "unknownFields"),
			cmpopts.IgnoreFields(routeguide.Point{}, "state", "sizeCache", "unknownFields"),
		}
		if diff := cmp.Diff(a, b, opts...); diff == "" {
			t.Errorf("got same responses: %#v", a)
		}
	}
}

func TestResponseDynamicRepeated(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/hello.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("Hello").ResponseDynamic()
	client := hello.NewGrpcTestServiceClient(ts.Conn())
	res, err := client.Hello(ctx, &hello.HelloRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hellos) == 0 {
		t.Error("invalid repeated field value")
	}
}

func TestResponseDynamicGenerated(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/hello.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	want := time.Now()
	opts := []GeneratorOption{
		Generator("*_time", func(req *Request) any {
			return want
		}),
	}
	ts.Method("Hello").ResponseDynamic(opts...)
	client := hello.NewGrpcTestServiceClient(ts.Conn())
	res, err := client.Hello(ctx, &hello.HelloRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.CreateTime.AsTime().UnixNano() != want.UnixNano() {
		t.Errorf("got %v\nwant %v", res.CreateTime.AsTime().UnixNano(), want.UnixNano())
	}
}

func TestResponseDynamicServer(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/hello.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	want := time.Now()
	opts := []GeneratorOption{
		Generator("*_time", func(req *Request) any {
			return want
		}),
	}
	ts.ResponseDynamic(opts...)
	client := hello.NewGrpcTestServiceClient(ts.Conn())
	res, err := client.Hello(ctx, &hello.HelloRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.CreateTime.AsTime().UnixNano() != want.UnixNano() {
		t.Errorf("got %v\nwant %v", res.CreateTime.AsTime().UnixNano(), want.UnixNano())
	}
}
