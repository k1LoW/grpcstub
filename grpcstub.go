package grpcstub

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/k1LoW/bufresolv"
	"github.com/k1LoW/protoresolv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

type serverStatus int

const (
	status_unknown serverStatus = iota
	status_start
	status_starting
	status_closing
	status_closed
)

const (
	HealthCheckService_DEFAULT  = "default"
	HealthCheckService_FLAPPING = "flapping"
)

var _ TB = (testing.TB)(nil)

type TB interface {
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
}

type Message map[string]any

type Request struct {
	Service string
	Method  string
	Headers metadata.MD
	Message Message
}

func (req *Request) String() string {
	var s []string
	s = append(s, fmt.Sprintf("%s/%s", req.Service, req.Method))
	if len(req.Headers) > 0 {
		var keys []string
		for k := range req.Headers {
			keys = append(keys, k)
		}
		sort.SliceStable(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})
		for _, k := range keys {
			s = append(s, fmt.Sprintf(`%s: %s`, k, strings.Join(req.Headers.Get(k), ", ")))
		}
	}
	s = append(s, "")
	if req.Message != nil {
		b, _ := json.MarshalIndent(req.Message, "", "  ")
		s = append(s, string(b))
	}
	return strings.Join(s, "\n") + "\n"
}

func newRequest(md protoreflect.MethodDescriptor, message Message) *Request {
	service, method := splitMethodFullName(md.FullName())
	return &Request{
		Service: service,
		Method:  method,
		Headers: metadata.MD{},
		Message: message,
	}
}

type Response struct {
	Headers  metadata.MD
	Messages []Message
	Trailers metadata.MD
	Status   *status.Status
}

// NewResponse returns a new empty response
func NewResponse() *Response {
	return &Response{
		Headers:  metadata.MD{},
		Messages: []Message{},
		Trailers: metadata.MD{},
		Status:   nil,
	}
}

type Server struct {
	matchers          []*matcher
	fds               linker.Files
	listener          net.Listener
	server            *grpc.Server
	tlsc              *tls.Config
	cacert            []byte
	cc                *grpc.ClientConn
	requests          []*Request
	unmatchedRequests []*Request
	healthCheck       bool
	disableReflection bool
	status            serverStatus
	prependOnce       bool
	t                 TB
	mu                sync.RWMutex
}

type matcher struct {
	matchFuncs []matchFunc
	handler    handlerFunc
	requests   []*Request
	t          TB
	mu         sync.RWMutex
}

type matchFunc func(req *Request) bool
type handlerFunc func(req *Request, md protoreflect.MethodDescriptor) *Response

// NewServer returns a new server with registered *grpc.Server
// protopath is a path of .proto files, import path directory or buf directory.
func NewServer(t TB, protopath string, opts ...Option) *Server {
	t.Helper()
	ctx := context.Background()
	c := &config{}
	if protopath != "" {
		if fi, err := os.Stat(protopath); err == nil && fi.IsDir() {
			if _, err := os.Stat(filepath.Join(protopath, "buf.yaml")); err == nil {
				opts = append(opts, BufDir(protopath))
			} else if _, err := os.Stat(filepath.Join(protopath, "buf.lock")); err == nil {
				opts = append(opts, BufDir(protopath))
			} else if _, err := os.Stat(filepath.Join(protopath, "buf.work.yaml")); err == nil {
				opts = append(opts, BufDir(protopath))
			} else {
				opts = append(opts, ImportPath(protopath))
			}
		} else {
			opts = append(opts, Proto(protopath))
		}
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			t.Fatal(err)
		}
	}
	s := &Server{
		t:                 t,
		healthCheck:       c.healthCheck,
		disableReflection: c.disableReflection,
	}
	if err := s.resolveProtos(ctx, c); err != nil {
		t.Fatal(err)
	}
	if c.useTLS {
		certificate, err := tls.X509KeyPair(c.cert, c.key)
		if err != nil {
			t.Fatal(err)
		}
		tlsc := &tls.Config{
			Certificates: []tls.Certificate{certificate},
		}
		creds := credentials.NewTLS(tlsc)
		s.tlsc = tlsc
		s.cacert = c.cacert
		s.server = grpc.NewServer(grpc.Creds(creds))
	} else {
		s.server = grpc.NewServer()
	}
	s.startServer()
	return s
}

// NewTLSServer returns a new server with registered secure *grpc.Server
func NewTLSServer(t TB, protopath string, cacert, cert, key []byte, opts ...Option) *Server {
	t.Helper()
	opts = append(opts, UseTLS(cacert, cert, key))
	return NewServer(t, protopath, opts...)
}

// Close shuts down *grpc.Server
func (s *Server) Close() {
	s.mu.Lock()
	s.status = status_closing
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.status = status_closed
		s.mu.Unlock()
	}()
	s.t.Helper()
	if s.listener == nil {
		s.t.Error("server is not started yet")
		return
	}
	if s.cc != nil {
		_ = s.cc.Close()
		s.cc = nil
	}
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()
	t := time.NewTimer(5 * time.Second)
	select {
	case <-done:
		if !t.Stop() {
			<-t.C
		}
	case <-t.C:
		s.server.Stop()
	}
}

// Addr returns server listener address
func (s *Server) Addr() string {
	s.t.Helper()
	if s.listener == nil {
		s.t.Error("server is not started yet")
		return ""
	}
	return s.listener.Addr().String()
}

// Conn returns *grpc.ClientConn which connects *grpc.Server.
func (s *Server) Conn() *grpc.ClientConn {
	s.t.Helper()
	if s.listener == nil {
		s.t.Error("server is not started yet")
		return nil
	}
	var creds credentials.TransportCredentials
	if s.tlsc == nil {
		creds = insecure.NewCredentials()
	} else {
		if s.cacert == nil {
			s.tlsc.InsecureSkipVerify = true
		} else {
			pool := x509.NewCertPool()
			if ok := pool.AppendCertsFromPEM(s.cacert); !ok {
				s.t.Fatal(errors.New("failed to append ca certs"))
			}
			s.tlsc.RootCAs = pool
		}
		creds = credentials.NewTLS(s.tlsc)
	}
	conn, err := grpc.Dial( //nolint:staticcheck
		s.listener.Addr().String(),
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		s.t.Error(err)
		return nil
	}
	s.cc = conn
	return conn
}

// ClientConn is alias of Conn
func (s *Server) ClientConn() *grpc.ClientConn {
	return s.Conn()
}

func (s *Server) startServer() {
	s.mu.Lock()
	s.status = status_starting
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.status = status_start
		s.mu.Unlock()
	}()
	s.t.Helper()
	if !s.disableReflection {
		reflection.Register(s.server)
	}
	s.registerServer()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.t.Error(err)
		return
	}
	s.listener = l
	go func() {
		_ = s.server.Serve(l)
	}()
}

// Match create request matcher with matchFunc (func(req *grpcstub.Request) bool).
func (s *Server) Match(fn func(req *Request) bool) *matcher {
	m := &matcher{
		matchFuncs: []matchFunc{fn},
		t:          s.t,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addMatcher(m)
	return m
}

// Match append matchFunc (func(req *grpcstub.Request) bool) to request matcher.
func (m *matcher) Match(fn func(req *Request) bool) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

// Service create request matcher using service.
func (s *Server) Service(service string) *matcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn := serviceMatchFunc(service)
	m := &matcher{
		matchFuncs: []matchFunc{fn},
		t:          s.t,
	}
	s.addMatcher(m)
	return m
}

// Service append request matcher using service.
func (m *matcher) Service(service string) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn := serviceMatchFunc(service)
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

// Servicef create request matcher using sprintf-ed service.
func (s *Server) Servicef(format string, a ...any) *matcher {
	return s.Service(fmt.Sprintf(format, a...))
}

// Servicef append request matcher using sprintf-ed service.
func (m *matcher) Servicef(format string, a ...any) *matcher {
	return m.Service(fmt.Sprintf(format, a...))
}

// Method create request matcher using method.
func (s *Server) Method(method string) *matcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn := methodMatchFunc(method)
	m := &matcher{
		matchFuncs: []matchFunc{fn},
		t:          s.t,
	}
	s.addMatcher(m)
	return m
}

// Method append request matcher using method.
func (m *matcher) Method(method string) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn := methodMatchFunc(method)
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

// Methodf create request matcher using sprintf-ed method.
func (s *Server) Methodf(format string, a ...any) *matcher {
	return s.Method(fmt.Sprintf(format, a...))
}

// Methodf append request matcher using sprintf-ed method.
func (m *matcher) Methodf(format string, a ...any) *matcher {
	return m.Method(fmt.Sprintf(format, a...))
}

// Header append handler which append header to response.
func (m *matcher) Header(key, value string) *matcher {
	prev := m.handler
	m.handler = func(req *Request, md protoreflect.MethodDescriptor) *Response {
		var res *Response
		if prev == nil {
			res = NewResponse()
		} else {
			res = prev(req, md)
		}
		res.Headers.Append(key, value)
		return res
	}
	return m
}

// Trailer append handler which append trailer to response.
func (m *matcher) Trailer(key, value string) *matcher {
	prev := m.handler
	m.handler = func(req *Request, md protoreflect.MethodDescriptor) *Response {
		var res *Response
		if prev == nil {
			res = NewResponse()
		} else {
			res = prev(req, md)
		}
		res.Trailers.Append(key, value)
		return res
	}
	return m
}

// Handler set handler
func (m *matcher) Handler(fn func(req *Request) *Response) {
	m.handler = func(req *Request, md protoreflect.MethodDescriptor) *Response {
		return fn(req)
	}
}

// Response set handler which return response.
func (m *matcher) Response(message any) *matcher {
	mm := map[string]any{}
	switch v := message.(type) {
	case map[string]any:
		mm = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			m.t.Fatalf("failed to convert message: %v", err)
		}
		if err := json.Unmarshal(b, &mm); err != nil {
			m.t.Fatalf("failed to convert message: %v", err)
		}
	}
	prev := m.handler
	m.handler = func(req *Request, md protoreflect.MethodDescriptor) *Response {
		var res *Response
		if prev == nil {
			res = NewResponse()
		} else {
			res = prev(req, md)
		}
		res.Messages = append(res.Messages, mm)
		return res
	}
	return m
}

// ResponseString set handler which return response.
func (m *matcher) ResponseString(message string) *matcher {
	mes := make(map[string]any)
	_ = json.Unmarshal([]byte(message), &mes)
	return m.Response(mes)
}

// ResponseStringf set handler which return sprintf-ed response.
func (m *matcher) ResponseStringf(format string, a ...any) *matcher {
	return m.ResponseString(fmt.Sprintf(format, a...))
}

// Status set handler which return response with status
func (m *matcher) Status(s *status.Status) *matcher {
	prev := m.handler
	m.handler = func(req *Request, md protoreflect.MethodDescriptor) *Response {
		var res *Response
		if prev == nil {
			res = NewResponse()
		} else {
			res = prev(req, md)
		}
		res.Status = s
		return res
	}
	return m
}

// Requests returns []*grpcstub.Request received by router.
func (s *Server) Requests() []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requests
}

// UnmatchedRequests returns []*grpcstub.Request received but not matched by router.
func (s *Server) UnmatchedRequests() []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unmatchedRequests
}

// ClearMatchers clear matchers.
func (s *Server) ClearMatchers() {
	s.matchers = nil
}

// Prepend prepend matcher.
func (s *Server) Prepend() *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prependOnce = true
	return s
}

// ClearRequests clear requests.
func (s *Server) ClearRequests() {
	s.requests = nil
	s.unmatchedRequests = nil
}

// Requests returns []*grpcstub.Request received by matcher.
func (m *matcher) Requests() []*Request {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requests
}

func (s *Server) addMatcher(m *matcher) {
	if s.prependOnce {
		s.matchers = append([]*matcher{m}, s.matchers...)
		s.prependOnce = false
		return
	}
	s.matchers = append(s.matchers, m)
}

func (s *Server) registerServer() {
	for _, fd := range s.fds {
		for i := 0; i < fd.Services().Len(); i++ {
			s.server.RegisterService(s.createServiceDesc(fd.Services().Get(i)), nil)
		}
	}
	if !s.healthCheck {
		return
	}
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(s.server, healthSrv)
	healthSrv.SetServingStatus(HealthCheckService_DEFAULT, healthpb.HealthCheckResponse_SERVING)
	go func() {
		status := healthpb.HealthCheckResponse_SERVING
		healthSrv.SetServingStatus(HealthCheckService_FLAPPING, status)
		for {
			s.mu.Lock()
			ss := s.status
			s.mu.Unlock()
			switch ss {
			case status_start, status_starting:
				if status == healthpb.HealthCheckResponse_SERVING {
					status = healthpb.HealthCheckResponse_NOT_SERVING
				} else {
					status = healthpb.HealthCheckResponse_SERVING
				}
				healthSrv.SetServingStatus(HealthCheckService_FLAPPING, status)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (s *Server) createServiceDesc(sd protoreflect.ServiceDescriptor) *grpc.ServiceDesc {
	gsd := &grpc.ServiceDesc{
		ServiceName: string(sd.FullName()),
		HandlerType: nil,
		Metadata:    sd.ParentFile().Name(),
	}

	mds := []protoreflect.MethodDescriptor{}
	for i := 0; i < sd.Methods().Len(); i++ {
		mds = append(mds, sd.Methods().Get(i))
	}

	gsd.Methods, gsd.Streams = s.createMethodDescs(mds)
	return gsd
}

func (s *Server) createMethodDescs(mds []protoreflect.MethodDescriptor) ([]grpc.MethodDesc, []grpc.StreamDesc) {
	var methods []grpc.MethodDesc
	var streams []grpc.StreamDesc
	for _, md := range mds {
		if !md.IsStreamingClient() && !md.IsStreamingServer() {
			method := grpc.MethodDesc{
				MethodName: string(md.Name()),
				Handler:    s.createUnaryHandler(md),
			}
			methods = append(methods, method)
		} else {
			stream := grpc.StreamDesc{
				StreamName:    string(md.Name()),
				Handler:       s.createStreamHandler(md),
				ServerStreams: md.IsStreamingServer(),
				ClientStreams: md.IsStreamingClient(),
			}
			streams = append(streams, stream)
		}
	}
	return methods, streams
}

func (s *Server) createUnaryHandler(md protoreflect.MethodDescriptor) func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		in := dynamicpb.NewMessage(md.Input())
		if err := dec(in); err != nil {
			return nil, err
		}
		m, err := MarshalProtoMessage(in)
		if err != nil {
			return nil, err
		}
		req := newRequest(md, m)
		h, ok := metadata.FromIncomingContext(ctx)
		if ok {
			req.Headers = h
		}

		var mes *dynamicpb.Message
		for _, m := range s.matchers {
			if !m.matchRequest(req) {
				continue
			}
			s.mu.Lock()
			s.requests = append(s.requests, req)
			s.mu.Unlock()
			m.mu.Lock()
			m.requests = append(m.requests, req)
			m.mu.Unlock()
			res := m.handler(req, md)
			for k, v := range res.Headers {
				for _, vv := range v {
					if err := grpc.SetHeader(ctx, metadata.Pairs(k, vv)); err != nil {
						return nil, err
					}
				}
			}
			for k, v := range res.Trailers {
				for _, vv := range v {
					if err := grpc.SetTrailer(ctx, metadata.Pairs(k, vv)); err != nil {
						return nil, err
					}
				}
			}
			if res.Status != nil && res.Status.Err() != nil {
				return nil, res.Status.Err()
			}
			mes = dynamicpb.NewMessage(md.Output())
			if len(res.Messages) > 0 {
				if err := UnmarshalProtoMessage(res.Messages[0], mes); err != nil {
					return nil, err
				}
			}
			return mes, nil
		}

		s.mu.Lock()
		s.unmatchedRequests = append(s.unmatchedRequests, req)
		s.mu.Unlock()
		return mes, status.Error(codes.NotFound, codes.NotFound.String())
	}
}

func (s *Server) createStreamHandler(md protoreflect.MethodDescriptor) func(srv any, stream grpc.ServerStream) error {
	switch {
	case !md.IsStreamingClient() && md.IsStreamingServer():
		return s.createServerStreamingHandler(md)
	case md.IsStreamingClient() && !md.IsStreamingServer():
		return s.createClientStreamingHandler(md)
	case md.IsStreamingClient() && md.IsStreamingServer():
		return s.createBidiStreamingHandler(md)
	default:
		return func(srv any, stream grpc.ServerStream) error {
			return nil
		}
	}
}

func (s *Server) createServerStreamingHandler(md protoreflect.MethodDescriptor) func(srv any, stream grpc.ServerStream) error {
	return func(srv any, stream grpc.ServerStream) error {
		in := dynamicpb.NewMessage(md.Input())
		if err := stream.RecvMsg(in); err != nil {
			return err
		}
		m, err := MarshalProtoMessage(in)
		if err != nil {
			return err
		}
		r := newRequest(md, m)
		h, ok := metadata.FromIncomingContext(stream.Context())
		if ok {
			r.Headers = h
		}
		for _, m := range s.matchers {
			if !m.matchRequest(r) {
				continue
			}
			m.mu.Lock()
			m.requests = append(m.requests, r)
			m.mu.Unlock()
			s.mu.Lock()
			s.requests = append(s.requests, r)
			s.mu.Unlock()
			res := m.handler(r, md)
			for k, v := range res.Headers {
				for _, vv := range v {
					if err := stream.SendHeader(metadata.Pairs(k, vv)); err != nil {
						return err
					}
				}
			}
			for k, v := range res.Trailers {
				for _, vv := range v {
					stream.SetTrailer(metadata.Pairs(k, vv))
				}
			}
			if res.Status != nil && res.Status.Err() != nil {
				return res.Status.Err()
			}
			if len(res.Messages) > 0 {
				for _, resm := range res.Messages {
					mes := dynamicpb.NewMessage(md.Output())
					if err := UnmarshalProtoMessage(resm, mes); err != nil {
						return err
					}
					if err := stream.SendMsg(mes); err != nil {
						return err
					}
				}
			}
			return nil
		}
		s.mu.Lock()
		s.unmatchedRequests = append(s.unmatchedRequests, r)
		s.mu.Unlock()
		return status.Error(codes.NotFound, codes.NotFound.String())
	}
}

func (s *Server) createClientStreamingHandler(md protoreflect.MethodDescriptor) func(srv any, stream grpc.ServerStream) error {
	return func(srv any, stream grpc.ServerStream) error {
		rs := []*Request{}
		for {
			in := dynamicpb.NewMessage(md.Input())
			err := stream.RecvMsg(in)
			if err == nil {
				m, err := MarshalProtoMessage(in)
				if err != nil {
					return err
				}
				r := newRequest(md, m)
				h, ok := metadata.FromIncomingContext(stream.Context())
				if ok {
					r.Headers = h
				}
				rs = append(rs, r)
				continue
			}

			if err != io.EOF {
				s.mu.Lock()
				s.unmatchedRequests = append(s.unmatchedRequests, rs...)
				s.mu.Unlock()
				return err
			}

			var mes *dynamicpb.Message
			for _, m := range s.matchers {
				if !m.matchRequest(rs...) {
					continue
				}
				s.mu.Lock()
				s.requests = append(s.requests, rs...)
				s.mu.Unlock()
				m.mu.Lock()
				m.requests = append(m.requests, rs...)
				m.mu.Unlock()
				last := rs[len(rs)-1]
				res := m.handler(last, md)
				if res.Status != nil && res.Status.Err() != nil {
					return res.Status.Err()
				}
				mes = dynamicpb.NewMessage(md.Output())
				if len(res.Messages) > 0 {
					if err := UnmarshalProtoMessage(res.Messages[0], mes); err != nil {
						return err
					}
				}
				for k, v := range res.Headers {
					for _, vv := range v {
						if err := stream.SendHeader(metadata.Pairs(k, vv)); err != nil {
							return err
						}
					}
				}
				for k, v := range res.Trailers {
					for _, vv := range v {
						stream.SetTrailer((metadata.Pairs(k, vv)))
					}
				}
				return stream.SendMsg(mes)
			}
			s.mu.Lock()
			s.unmatchedRequests = append(s.unmatchedRequests, rs...)
			s.mu.Unlock()
			return status.Error(codes.NotFound, codes.NotFound.String())
		}
	}
}

func (s *Server) createBidiStreamingHandler(md protoreflect.MethodDescriptor) func(srv any, stream grpc.ServerStream) error {
	return func(srv any, stream grpc.ServerStream) error {
		headerSent := false
	L:
		for {
			in := dynamicpb.NewMessage(md.Input())
			err := stream.RecvMsg(in)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			m, err := MarshalProtoMessage(in)
			if err != nil {
				return err
			}
			r := newRequest(md, m)
			h, ok := metadata.FromIncomingContext(stream.Context())
			if ok {
				r.Headers = h
			}
			for _, m := range s.matchers {
				if !m.matchRequest(r) {
					continue
				}
				s.mu.Lock()
				s.requests = append(s.requests, r)
				s.mu.Unlock()
				m.mu.Lock()
				m.requests = append(m.requests, r)
				m.mu.Unlock()
				res := m.handler(r, md)
				if !headerSent {
					for k, v := range res.Headers {
						for _, vv := range v {
							if err := stream.SendHeader(metadata.Pairs(k, vv)); err != nil {
								return err
							}
							headerSent = true
						}
					}
				}
				for k, v := range res.Trailers {
					for _, vv := range v {
						stream.SetTrailer(metadata.Pairs(k, vv))
					}
				}
				if res.Status != nil && res.Status.Err() != nil {
					return res.Status.Err()
				}
				if len(res.Messages) > 0 {
					for _, resm := range res.Messages {
						mes := dynamicpb.NewMessage(md.Output())
						if err := UnmarshalProtoMessage(resm, mes); err != nil {
							return err
						}
						if err := stream.SendMsg(mes); err != nil {
							return err
						}
					}
				}
				continue L
			}
			s.mu.Lock()
			s.unmatchedRequests = append(s.unmatchedRequests, r)
			s.mu.Unlock()
			return status.Error(codes.NotFound, codes.NotFound.String())
		}
	}
}

// MarshalProtoMessage marshals [proto.Message] to [Message].
func MarshalProtoMessage(pm protoreflect.ProtoMessage) (Message, error) {
	b, err := protojson.MarshalOptions{UseProtoNames: true, UseEnumNumbers: true, EmitUnpopulated: true}.Marshal(pm)
	if err != nil {
		return nil, err
	}
	m := Message{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// UnmarshalProtoMessage unmarshals [Message] to [proto.Message].
func UnmarshalProtoMessage(m Message, pm protoreflect.ProtoMessage) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := (protojson.UnmarshalOptions{}).Unmarshal(b, pm); err != nil {
		return err
	}
	return nil
}

func (m *matcher) matchRequest(rs ...*Request) bool {
	for _, r := range rs {
		for _, fn := range m.matchFuncs {
			if !fn(r) {
				return false
			}
		}
	}
	return true
}

func serviceMatchFunc(service string) matchFunc {
	return func(req *Request) bool {
		return req.Service == strings.TrimPrefix(service, "/")
	}
}

func methodMatchFunc(method string) matchFunc {
	return func(req *Request) bool {
		if !strings.Contains(method, "/") {
			return req.Method == method
		}
		splitted := strings.Split(strings.TrimPrefix(method, "/"), "/")
		s := strings.Join(splitted[:len(splitted)-1], "/")
		m := splitted[len(splitted)-1]
		return req.Service == s && req.Method == m
	}
}

func (s *Server) resolveProtos(ctx context.Context, c *config) error {
	pr, err := protoresolv.New(c.importPaths, protoresolv.Proto(c.protos...))
	if err != nil {
		return err
	}
	var bufresolvOpts []bufresolv.Option
	for _, dir := range c.bufDirs {
		bufresolvOpts = append(bufresolvOpts, bufresolv.BufDir(dir))
	}
	for _, config := range c.bufConfigs {
		bufresolvOpts = append(bufresolvOpts, bufresolv.BufConfig(config))
	}
	for _, lock := range c.bufLocks {
		bufresolvOpts = append(bufresolvOpts, bufresolv.BufLock(lock))
	}
	bufresolvOpts = append(bufresolvOpts, bufresolv.BufModule(c.bufModules...))
	br, err := bufresolv.New(bufresolvOpts...)
	if err != nil {
		return err
	}
	comp := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(protocompile.CompositeResolver([]protocompile.Resolver{
			pr, br,
		})),
	}
	protos := unique(slices.Concat(pr.Paths(), br.Paths()))
	fds, err := comp.Compile(ctx, protos...)
	if err != nil {
		return err
	}
	if err := registerFiles(fds); err != nil {
		return err
	}
	s.fds = fds
	return nil
}

func registerFiles(fds linker.Files) (err error) {
	for _, fd := range fds {
		// Skip registration of already registered descriptors
		if _, err := protoregistry.GlobalFiles.FindFileByPath(fd.Path()); !errors.Is(err, protoregistry.NotFound) {
			continue
		}
		// Skip registration of conflicted descriptors
		conflict := false
		rangeTopLevelDescriptors(fd, func(d protoreflect.Descriptor) {
			if _, err := protoregistry.GlobalFiles.FindDescriptorByName(d.FullName()); err == nil {
				conflict = true
			}
		})
		if conflict {
			continue
		}

		if err := protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
			return err
		}
	}
	return nil
}

// copy from google.golang.org/protobuf/reflect/protoregistry
func rangeTopLevelDescriptors(fd protoreflect.FileDescriptor, f func(protoreflect.Descriptor)) {
	eds := fd.Enums()
	for i := eds.Len() - 1; i >= 0; i-- {
		f(eds.Get(i))
		vds := eds.Get(i).Values()
		for i := vds.Len() - 1; i >= 0; i-- {
			f(vds.Get(i))
		}
	}
	mds := fd.Messages()
	for i := mds.Len() - 1; i >= 0; i-- {
		f(mds.Get(i))
	}
	xds := fd.Extensions()
	for i := xds.Len() - 1; i >= 0; i-- {
		f(xds.Get(i))
	}
	sds := fd.Services()
	for i := sds.Len() - 1; i >= 0; i-- {
		f(sds.Get(i))
	}
}

func splitMethodFullName(mn protoreflect.FullName) (string, string) {
	splitted := strings.Split(string(mn), ".")
	service := strings.Join(splitted[:len(splitted)-1], ".")
	method := splitted[len(splitted)-1]
	return service, method
}
