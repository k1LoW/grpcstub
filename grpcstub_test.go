package grpcstub

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/jhump/protoreflect/v2/grpcreflect"
	"github.com/k1LoW/grpcstub/testdata/bsr/protobuf/gen/go/pinger"
	"github.com/k1LoW/grpcstub/testdata/bsr/protobuf/gen/go/pinger/pingerconnect"
	"github.com/k1LoW/grpcstub/testdata/hello"
	"github.com/k1LoW/grpcstub/testdata/routeguide"
	"github.com/tenntenn/golden"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	ts.Match(func(req *Request) bool {
		return req.Method == "GetFeature"
	}).Response(map[string]any{"name": "hello"})

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
	ts.Service("routeguide.RouteGuide").Match(func(req *Request) bool {
		return req.Method == "GetFeature"
	}).Response(map[string]any{"name": "hello"})

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
	ts.Service("routeguide.RouteGuide").Response(map[string]any{"name": "hello"})

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
	ts.Method("GetFeature").Service("routeguide.RouteGuide").Response(map[string]any{"name": "hello"})

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
	ts.Service("routeguide.RouteGuide").Method("GetFeature").Response(map[string]any{"name": "hello"})

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
	ts.Method("GetFeature").Header("session", "XXXxxXXX").Header("size", "213").Response(map[string]any{"name": "hello"})

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
	ts.Method("GetFeature").Trailer("session", "XXXxxXXX").Trailer("size", "213").Response(map[string]any{"name": "hello"})

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
	ts.Method("GetFeature").Response(map[string]any{"name": "hello"})

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
		{"testdata/bsr/protobuf"},
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
			if len(svcs) == 0 {
				t.Error("no services")
			}
		})
	}
}

func TestTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		res      map[string]any
		wantTime time.Time
	}{
		{
			"empty is 0 of UNIX timestamp",
			map[string]any{
				"message": "hello",
				"num":     3,
				"hellos":  []string{"hello", "world"},
			},
			time.Unix(0, 0),
		},
		{
			"timestamppb.Timestamp",
			map[string]any{
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
	ts.Method("GetFeature").Response(map[string]any{"name": "hello", "location": map[string]any{"latitude": 10, "longitude": 13}})
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

func TestRequestStringer(t *testing.T) {
	tests := []struct {
		r *Request
	}{
		{
			&Request{
				Service: "helloworld.Greeter",
				Method:  "SayHello",
				Message: map[string]any{"name": "alice"},
				Headers: map[string][]string{"foo": {"bar", "barbar"}, "baz": {"qux"}},
			},
		},
		{
			&Request{
				Service: "helloworld.Greeter",
				Method:  "SayHello",
			},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := tt.r.String()
			f := fmt.Sprintf("request_stringer_%d", i)
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, "testdata", f, got)
				return
			}
			if diff := golden.Diff(t, "testdata", f, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestResponseAny(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("routeguide.RouteGuide").Method("GetFeature").Response(&routeguide.Feature{
		Name: "hello",
	})

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

func TestResponseAnyFields(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/hello.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Service("hello.GrpcTestService").Method("HelloFields").ResponseString(`{"field_bytes": "aGVsbG8="}`) // Base64 encoding to pass bytes type

	client := hello.NewGrpcTestServiceClient(ts.Conn())
	res, err := client.HelloFields(ctx, &hello.HelloFieldsRequest{
		FieldBytes: []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := res.FieldBytes
	if want := "hello"; string(got) != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestBufProtoRegistry(t *testing.T) {
	t.Run("Use buf.lock", func(t *testing.T) {
		ts := NewServer(t, "testdata/bsr/protobuf/pinger/pinger.proto", BufLock("testdata/bsr/protobuf/buf.lock"))
		t.Cleanup(func() {
			ts.Close()
		})
		ts.Service("pinger.PingerService").Method("Ping").Response(map[string]any{
			"message": "hello",
		})
	})

	t.Run("Use buf.yaml", func(t *testing.T) {
		ts := NewServer(t, "testdata/bsr/protobuf/pinger/pinger.proto", BufConfig("testdata/bsr/protobuf/buf.yaml"))
		t.Cleanup(func() {
			ts.Close()
		})
		ts.Service("pinger.PingerService").Method("Ping").Response(map[string]any{
			"message": "hello",
		})
	})

	t.Run("Specify modules", func(t *testing.T) {
		ts := NewServer(t, "testdata/bsr/protobuf/pinger/pinger.proto", BufModule("buf.build/bufbuild/protovalidate/tree/b983156c5e994cc9892e0ce3e64e17e0"))
		t.Cleanup(func() {
			ts.Close()
		})
		ts.Service("pinger.PingerService").Method("Ping").Response(map[string]any{
			"message": "hello",
		})
	})
}

func TestWithConnectClient(t *testing.T) {
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
	ts := NewTLSServer(t, "testdata/bsr/protobuf", cacert, cert, key)
	ts.Service("pinger.PingerService").Method("Ping").Response(&pinger.PingResponse{
		Message: "hello",
	})
	t.Cleanup(func() {
		ts.Close()
	})
	u := fmt.Sprintf("https://%s", ts.Addr())
	httpClient := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint: gosec
		},
	}
	client := pingerconnect.NewPingerServiceClient(httpClient, u, connect.WithGRPC())
	res, err := client.Ping(ctx, connect.NewRequest(&pinger.PingRequest{
		Message: "hello",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if want := "hello"; res.Msg.GetMessage() != want {
		t.Errorf("got %v\nwant %v", res.Msg.GetMessage(), want)
	}
}

func TestUnmarshalProtoMessage(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Match(func(req *Request) bool {
		return req.Method == "GetFeature"
	}).Handler(func(req *Request) *Response {
		m := &routeguide.Point{}
		if err := UnmarshalProtoMessage(req.Message, m); err != nil {
			t.Fatal(err)
		}
		if m.Latitude != 10 || m.Longitude != 13 {
			t.Errorf("got %v\nwant %v", m, &routeguide.Point{Latitude: 10, Longitude: 13})
		}
		return &Response{
			Messages: []Message{
				{"name": "hello"},
			},
		}
	})

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

func TestPrepend(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		ctx := context.Background()
		ts := NewServer(t, "testdata/route_guide.proto")
		t.Cleanup(func() {
			ts.Close()
		})
		ts.Service("routeguide.RouteGuide").Response(map[string]any{"name": "hello"})
		ts.Service("routeguide.RouteGuide").Response(map[string]any{"name": "world"})
		ts.Service("routeguide.RouteGuide").Response(map[string]any{"name": "!!!"})

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
	})

	t.Run("Prepend", func(t *testing.T) {
		ctx := context.Background()
		ts := NewServer(t, "testdata/route_guide.proto")
		t.Cleanup(func() {
			ts.Close()
		})
		ts.Service("routeguide.RouteGuide").Response(map[string]any{"name": "hello"})
		ts.Prepend().Service("routeguide.RouteGuide").Response(map[string]any{"name": "world"})
		ts.Service("routeguide.RouteGuide").Response(map[string]any{"name": "!!!"})

		client := routeguide.NewRouteGuideClient(ts.Conn())
		res, err := client.GetFeature(ctx, &routeguide.Point{
			Latitude:  10,
			Longitude: 13,
		})
		if err != nil {
			t.Fatal(err)
		}
		got := res.Name
		if want := "world"; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	})
}
