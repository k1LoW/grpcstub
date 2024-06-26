// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: pinger/pinger.proto

package pingerconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	pinger "github.com/k1LoW/grpcstub/testdata/bsr/protobuf/gen/go/pinger"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// PingerServiceName is the fully-qualified name of the PingerService service.
	PingerServiceName = "pinger.PingerService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// PingerServicePingProcedure is the fully-qualified name of the PingerService's Ping RPC.
	PingerServicePingProcedure = "/pinger.PingerService/Ping"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	pingerServiceServiceDescriptor    = pinger.File_pinger_pinger_proto.Services().ByName("PingerService")
	pingerServicePingMethodDescriptor = pingerServiceServiceDescriptor.Methods().ByName("Ping")
)

// PingerServiceClient is a client for the pinger.PingerService service.
type PingerServiceClient interface {
	Ping(context.Context, *connect.Request[pinger.PingRequest]) (*connect.Response[pinger.PingResponse], error)
}

// NewPingerServiceClient constructs a client for the pinger.PingerService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewPingerServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) PingerServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &pingerServiceClient{
		ping: connect.NewClient[pinger.PingRequest, pinger.PingResponse](
			httpClient,
			baseURL+PingerServicePingProcedure,
			connect.WithSchema(pingerServicePingMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// pingerServiceClient implements PingerServiceClient.
type pingerServiceClient struct {
	ping *connect.Client[pinger.PingRequest, pinger.PingResponse]
}

// Ping calls pinger.PingerService.Ping.
func (c *pingerServiceClient) Ping(ctx context.Context, req *connect.Request[pinger.PingRequest]) (*connect.Response[pinger.PingResponse], error) {
	return c.ping.CallUnary(ctx, req)
}

// PingerServiceHandler is an implementation of the pinger.PingerService service.
type PingerServiceHandler interface {
	Ping(context.Context, *connect.Request[pinger.PingRequest]) (*connect.Response[pinger.PingResponse], error)
}

// NewPingerServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewPingerServiceHandler(svc PingerServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	pingerServicePingHandler := connect.NewUnaryHandler(
		PingerServicePingProcedure,
		svc.Ping,
		connect.WithSchema(pingerServicePingMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/pinger.PingerService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case PingerServicePingProcedure:
			pingerServicePingHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedPingerServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedPingerServiceHandler struct{}

func (UnimplementedPingerServiceHandler) Ping(context.Context, *connect.Request[pinger.PingRequest]) (*connect.Response[pinger.PingResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("pinger.PingerService.Ping is not implemented"))
}
