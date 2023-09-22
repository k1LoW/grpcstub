package grpcstub

import (
	"context"
	"testing"

	"github.com/k1LoW/grpcstub/testdata/routeguide"
)

func TestClientStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RecordRoute").Response(map[string]any{"point_count": 2, "feature_count": 2, "distance": 10, "elapsed_time": 345})

	client := routeguide.NewRouteGuideClient(ts.Conn())
	stream, err := client.RecordRoute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c := 2
	for i := 0; i < c; i++ {
		if err := stream.Send(&routeguide.Point{
			Latitude:  int32(i + 10),
			Longitude: int32(i * i * 2),
		}); err != nil {
			t.Fatal(err)
		}
	}
	res, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatal(err)
	}

	{
		got := res.PointCount
		if want := int32(2); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}

	{
		got := len(ts.Requests())
		if want := 2; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}
