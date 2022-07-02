package grpcstub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/k1LoW/grpcstub/testdata/routeguide"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnary(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello", "location": map[string]interface{}{"latitude": 10, "longitude": 13}})

	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
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
		}
	}
}

func TestClientStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RecordRoute").Response(map[string]interface{}{"point_count": 2, "feature_count": 2, "distance": 10, "elapsed_time": 345})

	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
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

func TestServerStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("ListFeatures").Response(map[string]interface{}{"name": "hello"}).Response(map[string]interface{}{"name": "world"})

	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
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
			t.Error(err)
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

func TestBiStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RouteChat").Match(func(r *Request) bool {
		m, err := r.Message.Get("/message")
		if err != nil {
			return false
		}
		return strings.Contains(m.(string), "hello from client[0]")
	}).Response(map[string]interface{}{"location": nil, "message": "hello from server[0]"})
	ts.Method("RouteChat").Handler(func(r *Request) *Response {
		res := NewResponse()
		m, err := r.Message.Get("/message")
		if err != nil {
			res.Status = status.New(codes.Unknown, codes.Unknown.String())
			return res
		}
		mes := Message{}
		mes.Set("/message", strings.Replace(m.(string), "client", "server", 1))
		res.Messages = []Message{mes}
		return res
	})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)

	stream, err := client.RouteChat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	max := 5
	c := 0
	recvCount := 0
	var sendEnd, recvEnd bool
	for !(sendEnd && recvEnd) {
		if !sendEnd {
			if err := stream.SendMsg(&routeguide.RouteNote{
				Message: fmt.Sprintf("hello from client[%d]", c),
			}); err != nil {
				t.Error(err)
				sendEnd = true
			}
			c++
			if c == max {
				sendEnd = true
				if err := stream.CloseSend(); err != nil {
					t.Error(err)
				}
			}
		}

		if !recvEnd {
			if res, err := stream.Recv(); err != nil {
				if !errors.Is(err, io.EOF) {
					t.Error(err)
				}
				recvEnd = true
			} else {
				recvCount++
				got := res.Message
				if want := fmt.Sprintf("hello from server[%d]", recvCount-1); got != want {
					t.Errorf("got %v\nwant %v", got, want)
				}
			}
		}
	}
	if recvCount != max {
		t.Errorf("got %v\nwant %v", recvCount, max)
	}

	{
		got := len(ts.Requests())
		if want := max; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}

func TestServerMatch(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Match(func(r *Request) bool {
		return r.Method == "GetFeature"
	}).Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Name
	if want := "hello"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestMatcherMatch(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Match(func(r *Request) bool {
		return r.Method == "GetFeature"
	}).Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Name
	if want := "hello"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestServerService(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Name
	if want := "hello"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestMatcherService(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Service("routeguide.RouteGuide").Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Name
	if want := "hello"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestMatcherMethod(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Method("GetFeature").Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	res, err := client.GetFeature(ctx, &routeguide.Point{
		Latitude:  10,
		Longitude: 13,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Name
	if want := "hello"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestHeader(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Header("session", "XXXxxXXX").Header("size", "213").Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	var header metadata.MD
	if _, err := client.GetFeature(ctx, &routeguide.Point{}, grpc.Header(&header)); err != nil {
		t.Fatal(err)
	}
	{
		got := header.Get("session")
		if want := "XXXxxXXX"; got[0] != want {
			t.Errorf("got %v\nwant %v", got[0], want)
		}
	}
	{
		got := header.Get("size")
		if want := "213"; got[0] != want {
			t.Errorf("got %v\nwant %v", got[0], want)
		}
	}
}

func TestTrailer(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Trailer("session", "XXXxxXXX").Trailer("size", "213").Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	var trailer metadata.MD
	if _, err := client.GetFeature(ctx, &routeguide.Point{}, grpc.Trailer(&trailer)); err != nil {
		t.Fatal(err)
	}
	{
		got := trailer.Get("session")
		if want := "XXXxxXXX"; got[0] != want {
			t.Errorf("got %v\nwant %v", got[0], want)
		}
	}
	{
		got := trailer.Get("size")
		if want := "213"; got[0] != want {
			t.Errorf("got %v\nwant %v", got[0], want)
		}
	}
}

func TestResponseHeader(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, []string{}, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello"})
	conn, err := ts.Conn()
	if err != nil {
		t.Fatal(err)
	}
	client := routeguide.NewRouteGuideClient(conn)
	ctx = metadata.AppendToOutgoingContext(ctx, "authentication", "XXXXxxxxXXXX")
	if _, err := client.GetFeature(ctx, &routeguide.Point{}); err != nil {
		t.Fatal(err)
	}
	r := ts.Requests()[0]
	got := r.Headers.Get("authentication")
	if want := "XXXXxxxxXXXX"; got[0] != want {
		t.Errorf("got %v\nwant %v", got[0], want)
	}
}
