// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: pinger/pinger.proto

package pinger

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	PingerService_Ping_FullMethodName = "/pinger.PingerService/Ping"
)

// PingerServiceClient is the client API for PingerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PingerServiceClient interface {
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
}

type pingerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPingerServiceClient(cc grpc.ClientConnInterface) PingerServiceClient {
	return &pingerServiceClient{cc}
}

func (c *pingerServiceClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := c.cc.Invoke(ctx, PingerService_Ping_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PingerServiceServer is the server API for PingerService service.
// All implementations must embed UnimplementedPingerServiceServer
// for forward compatibility
type PingerServiceServer interface {
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	mustEmbedUnimplementedPingerServiceServer()
}

// UnimplementedPingerServiceServer must be embedded to have forward compatible implementations.
type UnimplementedPingerServiceServer struct {
}

func (UnimplementedPingerServiceServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedPingerServiceServer) mustEmbedUnimplementedPingerServiceServer() {}

// UnsafePingerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PingerServiceServer will
// result in compilation errors.
type UnsafePingerServiceServer interface {
	mustEmbedUnimplementedPingerServiceServer()
}

func RegisterPingerServiceServer(s grpc.ServiceRegistrar, srv PingerServiceServer) {
	s.RegisterService(&PingerService_ServiceDesc, srv)
}

func _PingerService_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PingerServiceServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PingerService_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PingerServiceServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PingerService_ServiceDesc is the grpc.ServiceDesc for PingerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PingerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pinger.PingerService",
	HandlerType: (*PingerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _PingerService_Ping_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pinger/pinger.proto",
}
