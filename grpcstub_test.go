package grpcstub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jhump/protoreflect/v2/grpcreflect"
	"github.com/k1LoW/grpcstub/testdata/hello"
	"github.com/k1LoW/grpcstub/testdata/routeguide"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUnary(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello", "location": map[string]interface{}{"latitude": 10, "longitude": 13}})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello", "location": map[string]interface{}{"latitude": 99, "longitude": 99}})

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

func TestServerStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("ListFeatures").Response(map[string]interface{}{"name": "hello"}).Response(map[string]interface{}{"name": "world"})

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

func TestClientStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RecordRoute").Response(map[string]interface{}{"point_count": 2, "feature_count": 2, "distance": 10, "elapsed_time": 345})

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

func TestBiStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RouteChat").Match(func(r *Request) bool {
		m, ok := r.Message["message"]
		if !ok {
			return false
		}
		return strings.Contains(m.(string), "hello from client[0]")
	}).Header("hello", "header").
		Response(map[string]interface{}{"location": nil, "message": "hello from server[0]"})
	ts.Method("RouteChat").
		Header("hello", "header").
		Handler(func(r *Request) *Response {
			res := NewResponse()
			m, ok := r.Message["message"]
			if !ok {
				res.Status = status.New(codes.Unknown, codes.Unknown.String())
				return res
			}
			mes := Message{}
			mes["message"] = strings.Replace(m.(string), "client", "server", 1)
			res.Messages = []Message{mes}
			return res
		})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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

func TestAddr(t *testing.T) {
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	got := ts.Addr()
	if !strings.HasPrefix(got, "127.0.0.1:") {
		t.Errorf("got %v\nwant 127.0.0.1:*", got)
	}
}

func TestServerMatch(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Match(func(r *Request) bool {
		return r.Method == "GetFeature"
	}).Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Match(func(r *Request) bool {
		return r.Method == "GetFeature"
	}).Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Service("routeguide.RouteGuide").Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Method("GetFeature").Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Header("session", "XXXxxXXX").Header("size", "213").Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Trailer("session", "XXXxxXXX").Trailer("size", "213").Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Response(map[string]interface{}{"name": "hello"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
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

func TestStatusUnary(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("GetFeature").Status(status.New(codes.Aborted, "aborted"))
	client := routeguide.NewRouteGuideClient(ts.Conn())

	_, err := client.GetFeature(ctx, &routeguide.Point{})
	if err == nil {
		t.Error("want error")
		return
	}

	s, ok := status.FromError(err)
	if !ok {
		t.Error("want status.Status")
		return
	}
	{
		got := s.Code()
		if want := codes.Aborted; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	{
		got := s.Message()
		if want := "aborted"; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}

func TestStatusServerStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("ListFeatures").Status(status.New(codes.Aborted, "aborted"))

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

	_, err = stream.Recv()
	if err == nil {
		t.Error("want error")
	}
	s, ok := status.FromError(err)
	if !ok {
		t.Error("want status.Status")
		return
	}
	{
		got := s.Code()
		if want := codes.Aborted; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	{
		got := s.Message()
		if want := "aborted"; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}

func TestStatusClientStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RecordRoute").Status(status.New(codes.Aborted, "aborted"))

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
	_, err = stream.CloseAndRecv()
	if err == nil {
		t.Error("want error")
		return
	}

	s, ok := status.FromError(err)
	if !ok {
		t.Error("want status.Status")
		return
	}
	{
		got := s.Code()
		if want := codes.Aborted; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	{
		got := s.Message()
		if want := "aborted"; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}

func TestStatusBiStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RouteChat").Header("hello", "header").Trailer("hello", "trailer").Status(status.New(codes.Aborted, "aborted"))

	client := routeguide.NewRouteGuideClient(ts.Conn())
	stream, err := client.RouteChat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.SendMsg(&routeguide.RouteNote{
		Message: "hello from client",
	}); err != nil {
		t.Fatal(err)
	}
	_, err = stream.Recv()
	if err == nil {
		t.Error("want error")
		return
	}

	s, ok := status.FromError(err)
	if !ok {
		t.Error("want status.Status")
		return
	}
	{
		got := s.Code()
		if want := codes.Aborted; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	{
		got := s.Message()
		if want := "aborted"; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
	h, err := stream.Header()
	if err != nil {
		t.Error(err)
	}
	{
		got := h.Get("hello")
		want := []string{"header"}
		if diff := cmp.Diff(got, want, nil); diff != "" {
			t.Errorf("%s", diff)
		}
	}
	{
		got := stream.Trailer().Get("hello")
		want := []string{"trailer"}
		if diff := cmp.Diff(got, want, nil); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}

func TestLoadProto(t *testing.T) {
	tests := []struct {
		proto string
	}{
		{"testdata/route_guide.proto"},
		{"testdata/hello.proto"},
		{"testdata/*.proto"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.proto, func(t *testing.T) {
			ts := NewServer(t, tt.proto)
			t.Cleanup(func() {
				ts.Close()
			})
			cc := ts.ClientConn()
			client := grpcreflect.NewClientAuto(ctx, cc)
			svcs, err := client.ListServices()
			if err != nil {
				t.Fatal(err)
			}
			resolver := client.AsResolver()
			for _, svc := range svcs {
				sd, err := resolver.FindServiceByName(svc)
				if err != nil {
					t.Fatal(err)
				}
				mds := sd.Methods()
				for i := 0; i < mds.Len(); i++ {
					md := mds.Get(i)
					_, err := resolver.FindMethodByName(md.FullName())
					if err != nil {
						t.Errorf("method not found: %s", md.FullName())
					}
				}
			}
		})
	}
}

func TestTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		res      map[string]interface{}
		wantTime time.Time
	}{
		{
			"empty is 0 of UNIX timestamp",
			map[string]interface{}{
				"message": "hello",
				"num":     3,
				"hellos":  []string{"hello", "world"},
			},
			time.Unix(0, 0),
		},
		{
			"timestamppb.Timestamp",
			map[string]interface{}{
				"message":     "hello",
				"num":         3,
				"hellos":      []string{"hello", "world"},
				"create_time": now.Format(time.RFC3339Nano),
			},
			now,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewServer(t, "testdata/hello.proto")
			t.Cleanup(func() {
				ts.Close()
			})
			ts.Method("Hello").Response(tt.res)
			client := hello.NewGrpcTestServiceClient(ts.Conn())
			got, err := client.Hello(ctx, &hello.HelloRequest{
				Name:        "alice",
				Num:         35,
				RequestTime: timestamppb.New(now),
			})
			if err != nil {
				t.Error(err)
				return
			}
			if got.CreateTime.AsTime().Unix() != tt.wantTime.Unix() {
				t.Errorf("got %v\nwant %v", got.CreateTime.AsTime(), tt.wantTime)
			}
		})
	}
}

func TestTLSServer(t *testing.T) {
	ctx := context.Background()
	cacert, err := os.ReadFile("testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	cert, err := os.ReadFile("testdata/cert.pem")
	if err != nil {
		t.Fatal(err)
	}
	key, err := os.ReadFile("testdata/key.pem")
	if err != nil {
		t.Fatal(err)
	}
	ts := NewTLSServer(t, "testdata/route_guide.proto", cacert, cert, key)
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
		}
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		enable  bool
		wantErr bool
	}{
		{true, false},
		{false, true},
	}
	ctx := context.Background()
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			var ts *Server
			if tt.enable {
				ts = NewServer(t, "testdata/*.proto", EnableHealthCheck())
			} else {
				ts = NewServer(t, "testdata/*.proto")
			}
			t.Cleanup(func() {
				ts.Close()
			})
			client := healthpb.NewHealthClient(ts.ClientConn())
			_, err := client.Check(ctx, &healthpb.HealthCheckRequest{
				Service: HealthCheckService_DEFAULT,
			})
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got error: %s", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want error")
			}
		})
	}
}

func TestReflection(t *testing.T) {
	tests := []struct {
		disableReflection bool
		wantErr           bool
	}{
		{false, false},
		{true, true},
	}
	proto := "testdata/route_guide.proto"
	ctx := context.Background()
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			opts := []Option{}
			if tt.disableReflection {
				opts = append(opts, DisableReflection())
			}
			ts := NewServer(t, proto, opts...)
			t.Cleanup(func() {
				ts.Close()
			})
			cc := ts.ClientConn()
			client := grpcreflect.NewClientAuto(ctx, cc)
			_, err := client.ListServices()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got error: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want error")
			}
		})
	}
}
