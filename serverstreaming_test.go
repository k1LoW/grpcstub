package grpcstub

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/k1LoW/grpcstub/testdata/routeguide"
)

func TestServerStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("ListFeatures").Response(map[string]any{"name": "hello"}).Response(map[string]any{"name": "world"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
	stream, err := client.ListFeatures(ctx, &routeguide.Rectangle{
		Lo: &routeguide.Point{
			Latitude:  int32(10),
			Longitude: int32(2),
		},
		Hi: &routeguide.Point{
			Latitude:  int32(20),
			Longitude: int32(7),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	c := 0
	for {
		res, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch c {
		case 0:
			got := res.Name
			if want := "hello"; got != want {
				t.Errorf("got %v\nwant %v", got, want)
			}
		case 1:
			got := res.Name
			if want := "world"; got != want {
				t.Errorf("got %v\nwant %v", got, want)
			}
		default:
			t.Errorf("recv messages got %v\nwant %v", c+1, 2)
		}
		c++
	}

	{
		got := len(ts.Requests())
		if want := 1; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}

func TestServerStreamingUnmatched(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("ListFeatures").Match(func(req *Request) bool {
		return false
	}).Response(map[string]any{"name": "hello"}).Response(map[string]any{"name": "world"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
	stream, err := client.ListFeatures(ctx, &routeguide.Rectangle{
		Lo: &routeguide.Point{
			Latitude:  int32(10),
			Longitude: int32(2),
		},
		Hi: &routeguide.Point{
			Latitude:  int32(20),
			Longitude: int32(7),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := stream.Recv(); err == nil || errors.Is(err, io.EOF) {
		t.Error("want error")
	}

	{
		got := len(ts.Requests())
		if want := 0; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	{
		got := len(ts.UnmatchedRequests())
		if want := 1; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}
