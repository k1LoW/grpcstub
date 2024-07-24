package grpcstub

import (
	"context"
	"testing"

	"github.com/k1LoW/grpcstub/testdata/routeguide"
)

func TestUnary(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]any{"name": "hello", "location": map[string]any{"latitude": 10, "longitude": 13}})
	ts.Method("GetFeature").Response(map[string]any{"name": "hello", "location": map[string]any{"latitude": 99, "longitude": 99}})

	client := routeguide.NewRouteGuideClient(ts.Conn())
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	{
		got := res.Name
		if want := "hello"; got != want {
			t.Errorf("got %v\nwant %v", got, want)
			return
		}
	}
	{
		got := res.Location.Latitude
		if want := int32(10); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}

	{
		got := len(ts.Requests())
		if want := 1; got != want {
			t.Errorf("got %v\nwant %v", got, want)
			return
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

func TestUnaryUnmatched(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Match(func(req *Request) bool {
		return false
	}).Response(map[string]any{"name": "hello", "location": map[string]any{"latitude": 10, "longitude": 13}})

	client := routeguide.NewRouteGuideClient(ts.Conn())
	_, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err == nil {
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
