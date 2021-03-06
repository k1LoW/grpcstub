package grpcstub

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb" //nolint
	"github.com/golang/protobuf/proto"  //nolint
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mattn/go-jsonpointer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type Message map[string]interface{}

func (m Message) Get(pointer string) (interface{}, error) {
	return jsonpointer.Get(m, pointer)
}

func (m Message) Has(pointer string) bool {
	return jsonpointer.Has(m, pointer)
}

func (m Message) Set(pointer string, value interface{}) error {
	return jsonpointer.Set(m, pointer, value)
}

type Request struct {
	Service string
	Method  string
	Headers metadata.MD
	Message Message
}

func newRequest(service, method string, message Message) *Request {
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
	matchers []*matcher
	fds      []*desc.FileDescriptor
	listener net.Listener
	server   *grpc.Server
	tlsc     *tls.Config
	cacert   []byte
	cc       *grpc.ClientConn
	requests []*Request
	t        *testing.T
	mu       sync.RWMutex
}

type matcher struct {
	matchFuncs []matchFunc
	handler    handlerFunc
	requests   []*Request
	mu         sync.RWMutex
}

type matchFunc func(r *Request) bool
type handlerFunc func(r *Request) *Response

// NewServer returns a new server with registered *grpc.Server
func NewServer(t *testing.T, importPaths []string, protos ...string) *Server {
	t.Helper()
	fds, err := descriptorFromFiles(importPaths, protos...)
	if err != nil {
		t.Error(err)
		return nil
	}
	s := &Server{
		fds:    fds,
		server: grpc.NewServer(),
		t:      t,
	}
	s.startServer()
	return s
}

// NewTLSServer returns a new server with registered secure *grpc.Server
func NewTLSServer(t *testing.T, cacert, cert, key []byte, importPaths []string, protos ...string) *Server {
	t.Helper()
	fds, err := descriptorFromFiles(importPaths, protos...)
	if err != nil {
		t.Error(err)
		return nil
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	tlsc := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}
	creds := credentials.NewTLS(tlsc)
	s := &Server{
		fds:    fds,
		tlsc:   tlsc,
		cacert: cacert,
		server: grpc.NewServer(grpc.Creds(creds)),
		t:      t,
	}
	s.startServer()
	return s
}

// Close shuts down *grpc.Server
func (s *Server) Close() {
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
	conn, err := grpc.Dial(
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
	s.t.Helper()
	reflection.Register(s.server)
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

// Match create request matcher with matchFunc (func(r *grpcstub.Request) bool).
func (s *Server) Match(fn func(r *Request) bool) *matcher {
	m := &matcher{
		matchFuncs: []matchFunc{fn},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matchers = append(s.matchers, m)
	return m
}

// Match append matchFunc (func(r *grpcstub.Request) bool) to request matcher.
func (m *matcher) Match(fn func(r *Request) bool) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

// Method create request matcher using service.
func (s *Server) Service(service string) *matcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn := serviceMatchFunc(service)
	m := &matcher{
		matchFuncs: []matchFunc{fn},
	}
	s.matchers = append(s.matchers, m)
	return m
}

// Method append request matcher using service.
func (m *matcher) Service(service string) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn := serviceMatchFunc(service)
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

// Method create request matcher using method.
func (s *Server) Method(method string) *matcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn := methodMatchFunc(method)
	m := &matcher{
		matchFuncs: []matchFunc{fn},
	}
	s.matchers = append(s.matchers, m)
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

// Header append handler which append header to response.
func (m *matcher) Header(key, value string) *matcher {
	var fn handlerFunc
	if m.handler == nil {
		fn = func(r *Request) *Response {
			res := NewResponse()
			res.Headers.Append(key, value)
			return res
		}
	} else {
		prev := m.handler
		fn = func(r *Request) *Response {
			res := prev(r)
			res.Headers.Append(key, value)
			return res
		}
	}
	m.handler = fn
	return m
}

// Trailer append handler which append trailer to response.
func (m *matcher) Trailer(key, value string) *matcher {
	var fn handlerFunc
	if m.handler == nil {
		fn = func(r *Request) *Response {
			res := NewResponse()
			res.Trailers.Append(key, value)
			return res
		}
	} else {
		prev := m.handler
		fn = func(r *Request) *Response {
			res := prev(r)
			res.Trailers.Append(key, value)
			return res
		}
	}
	m.handler = fn
	return m
}

// Handler set handler
func (m *matcher) Handler(fn func(r *Request) *Response) {
	m.handler = fn
}

// Response set hander which return response.
func (m *matcher) Response(message map[string]interface{}) *matcher {
	var fn handlerFunc
	if m.handler == nil {
		fn = func(r *Request) *Response {
			res := NewResponse()
			res.Messages = append(res.Messages, message)
			return res
		}
	} else {
		prev := m.handler
		fn = func(r *Request) *Response {
			res := prev(r)
			res.Messages = append(res.Messages, message)
			return res
		}
	}
	m.handler = fn
	return m
}

// ResponseString set hander which return response.
func (m *matcher) ResponseString(message string) *matcher {
	mes := make(map[string]interface{})
	_ = json.Unmarshal([]byte(message), &mes)
	return m.Response(mes)
}

// Status set status which return response.
func (m *matcher) Status(s *status.Status) *matcher {
	var fn handlerFunc
	if m.handler == nil {
		fn = func(r *Request) *Response {
			res := NewResponse()
			res.Status = s
			return res
		}
	} else {
		prev := m.handler
		fn = func(r *Request) *Response {
			res := prev(r)
			res.Status = s
			return res
		}
	}
	m.handler = fn
	return m
}

// Requests returns []*grpcstub.Request received by router.
func (s *Server) Requests() []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requests
}

// Requests returns []*grpcstub.Request received by matcher.
func (m *matcher) Requests() []*Request {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requests
}

func (s *Server) registerServer() {
	var sds []*grpc.ServiceDesc
	for _, fd := range s.fds {
		for _, sd := range fd.GetServices() {
			sds = append(sds, s.createServiceDesc(sd))
		}
	}
	for _, sd := range sds {
		s.server.RegisterService(sd, nil)
	}
}

func (s *Server) createServiceDesc(sd *desc.ServiceDescriptor) *grpc.ServiceDesc {
	gsd := &grpc.ServiceDesc{
		ServiceName: sd.GetFullyQualifiedName(),
		HandlerType: nil,
		Metadata:    sd.GetFile().GetName(),
	}

	gsd.Methods, gsd.Streams = s.createMethodDescs(sd.GetMethods())
	return gsd
}

func (s *Server) createMethodDescs(mds []*desc.MethodDescriptor) ([]grpc.MethodDesc, []grpc.StreamDesc) {
	var methods []grpc.MethodDesc
	var streams []grpc.StreamDesc
	for _, md := range mds {
		if !md.IsClientStreaming() && !md.IsServerStreaming() {
			method := grpc.MethodDesc{
				MethodName: md.GetName(),
				Handler:    s.createUnaryHandler(md),
			}
			methods = append(methods, method)
		} else {
			stream := grpc.StreamDesc{
				StreamName:    md.GetName(),
				Handler:       s.createStreamHandler(md),
				ServerStreams: md.IsServerStreaming(),
				ClientStreams: md.IsClientStreaming(),
			}
			streams = append(streams, stream)
		}
	}
	return methods, streams
}

func (s *Server) createUnaryHandler(md *desc.MethodDescriptor) func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
		msgFactory := dynamic.NewMessageFactoryWithDefaults()
		in := msgFactory.NewMessage(md.GetInputType())
		if err := dec(in); err != nil {
			return nil, err
		}
		b := new(bytes.Buffer)
		marshaler := jsonpb.Marshaler{
			OrigName: true,
		}
		if err := marshaler.Marshal(b, in); err != nil {
			return nil, err
		}
		m := Message{}
		if err := json.Unmarshal(b.Bytes(), &m); err != nil {
			return nil, err
		}
		r := newRequest(md.GetService().GetFullyQualifiedName(), md.GetName(), m)
		h, ok := metadata.FromIncomingContext(ctx)
		if ok {
			r.Headers = h
		}
		s.mu.Lock()
		s.requests = append(s.requests, r)
		s.mu.Unlock()

		var mes proto.Message
		for _, m := range s.matchers {
			match := true
			for _, fn := range m.matchFuncs {
				if !fn(r) {
					match = false
				}
			}
			if match {
				m.mu.Lock()
				m.requests = append(m.requests, r)
				m.mu.Unlock()
				res := m.handler(r)
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
				mes = msgFactory.NewMessage(md.GetOutputType())
				if len(res.Messages) > 0 {
					b, err := json.Marshal(res.Messages[0])
					if err != nil {
						return nil, err
					}
					if err := json.Unmarshal(b, mes); err != nil {
						return nil, err
					}
				}
				return mes, nil
			}
		}

		return mes, status.Error(codes.NotFound, codes.NotFound.String())
	}
}

func (s *Server) createStreamHandler(md *desc.MethodDescriptor) func(srv interface{}, stream grpc.ServerStream) error {
	switch {
	case !md.IsClientStreaming() && md.IsServerStreaming():
		return s.createServerStreamingHandler(md)
	case md.IsClientStreaming() && !md.IsServerStreaming():
		return s.createClientStreamingHandler(md)
	case md.IsClientStreaming() && md.IsServerStreaming():
		return s.createBiStreamingHandler(md)
	default:
		return func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}
	}
}

func (s *Server) createServerStreamingHandler(md *desc.MethodDescriptor) func(srv interface{}, stream grpc.ServerStream) error {
	return func(srv interface{}, stream grpc.ServerStream) error {
		msgFactory := dynamic.NewMessageFactoryWithDefaults()
		in := msgFactory.NewMessage(md.GetInputType())
		if err := stream.RecvMsg(in); err != nil {
			return err
		}
		b := new(bytes.Buffer)
		marshaler := jsonpb.Marshaler{
			OrigName: true,
		}
		if err := marshaler.Marshal(b, in); err != nil {
			return err
		}
		m := Message{}
		if err := json.Unmarshal(b.Bytes(), &m); err != nil {
			return err
		}
		r := newRequest(md.GetService().GetFullyQualifiedName(), md.GetName(), m)
		h, ok := metadata.FromIncomingContext(stream.Context())
		if ok {
			r.Headers = h
		}
		s.mu.Lock()
		s.requests = append(s.requests, r)
		s.mu.Unlock()
		for _, m := range s.matchers {
			match := true
			for _, fn := range m.matchFuncs {
				if !fn(r) {
					match = false
				}
			}
			if match {
				m.mu.Lock()
				m.requests = append(m.requests, r)
				m.mu.Unlock()
				res := m.handler(r)
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
						mes := msgFactory.NewMessage(md.GetOutputType())
						b, err := json.Marshal(resm)
						if err != nil {
							return err
						}
						if err := json.Unmarshal(b, mes); err != nil {
							return err
						}
						if err := stream.SendMsg(mes); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}
}

func (s *Server) createClientStreamingHandler(md *desc.MethodDescriptor) func(srv interface{}, stream grpc.ServerStream) error {
	return func(srv interface{}, stream grpc.ServerStream) error {
		rs := []*Request{}
		for {
			msgFactory := dynamic.NewMessageFactoryWithDefaults()
			in := msgFactory.NewMessage(md.GetInputType())
			err := stream.RecvMsg(in)
			if err == nil {
				b := new(bytes.Buffer)
				marshaler := jsonpb.Marshaler{
					OrigName: true,
				}
				if err := marshaler.Marshal(b, in); err != nil {
					return err
				}
				m := Message{}
				if err := json.Unmarshal(b.Bytes(), &m); err != nil {
					return err
				}
				r := newRequest(md.GetService().GetFullyQualifiedName(), md.GetName(), m)
				h, ok := metadata.FromIncomingContext(stream.Context())
				if ok {
					r.Headers = h
				}
				s.mu.Lock()
				s.requests = append(s.requests, r)
				s.mu.Unlock()
				rs = append(rs, r)
			}
			if err == io.EOF {
				var mes proto.Message
				for _, r := range rs {
					for _, m := range s.matchers {
						match := true
						for _, fn := range m.matchFuncs {
							if !fn(r) {
								match = false
							}
						}
						if match {
							m.mu.Lock()
							m.requests = append(m.requests, r)
							m.mu.Unlock()
							res := m.handler(r)
							if res.Status != nil && res.Status.Err() != nil {
								return res.Status.Err()
							}
							mes = msgFactory.NewMessage(md.GetOutputType())
							if len(res.Messages) > 0 {
								b, err := json.Marshal(res.Messages[0])
								if err != nil {
									return err
								}
								if err := json.Unmarshal(b, mes); err != nil {
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
					}
				}
				return status.Error(codes.NotFound, codes.NotFound.String())
			}
		}
	}
}

func (s *Server) createBiStreamingHandler(md *desc.MethodDescriptor) func(srv interface{}, stream grpc.ServerStream) error {
	return func(srv interface{}, stream grpc.ServerStream) error {
		headerSent := false
	L:
		for {
			msgFactory := dynamic.NewMessageFactoryWithDefaults()
			in := msgFactory.NewMessage(md.GetInputType())
			err := stream.RecvMsg(in)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			b := new(bytes.Buffer)
			marshaler := jsonpb.Marshaler{
				OrigName: true,
			}
			if err := marshaler.Marshal(b, in); err != nil {
				return err
			}
			m := Message{}
			if err := json.Unmarshal(b.Bytes(), &m); err != nil {
				return err
			}
			r := newRequest(md.GetService().GetFullyQualifiedName(), md.GetName(), m)
			h, ok := metadata.FromIncomingContext(stream.Context())
			if ok {
				r.Headers = h
			}
			s.mu.Lock()
			s.requests = append(s.requests, r)
			s.mu.Unlock()
			for _, m := range s.matchers {
				match := true
				for _, fn := range m.matchFuncs {
					if !fn(r) {
						match = false
					}
				}
				if match {
					m.mu.Lock()
					m.requests = append(m.requests, r)
					m.mu.Unlock()
					res := m.handler(r)
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
							mes := msgFactory.NewMessage(md.GetOutputType())
							b, err := json.Marshal(resm)
							if err != nil {
								return err
							}
							if err := json.Unmarshal(b, mes); err != nil {
								return err
							}
							if err := stream.SendMsg(mes); err != nil {
								return err
							}
						}
					}
					continue L
				}
			}
			return status.Error(codes.NotFound, codes.NotFound.String())
		}
	}
}

func descriptorFromFiles(importPaths []string, protos ...string) ([]*desc.FileDescriptor, error) {
	protos, err := protoparse.ResolveFilenames(importPaths, protos...)
	if err != nil {
		return nil, err
	}
	importPaths, protos, accessor, err := resolvePaths(importPaths, protos...)
	if err != nil {
		return nil, err
	}
	p := protoparse.Parser{
		ImportPaths:           importPaths,
		InferImportPaths:      len(importPaths) == 0,
		IncludeSourceCodeInfo: true,
		Accessor:              accessor,
	}
	fds, err := p.ParseFiles(protos...)
	if err != nil {
		return nil, err
	}
	if err := registerFileDescriptors(fds); err != nil {
		return nil, err
	}

	return fds, nil
}

func resolvePaths(importPaths []string, protos ...string) ([]string, []string, func(filename string) (io.ReadCloser, error), error) {
	var prefix string
	resolvedIPaths := []string{}
	resolvedProtos := []string{}
	for _, p := range importPaths {
		d, b := filepath.Split(p)
		if d != "" && prefix == "" {
			prefix = d
		}
		resolvedIPaths = append(resolvedIPaths, b)
	}
	for _, p := range protos {
		d, b := filepath.Split(p)
		if d != "" && prefix == "" {
			prefix = d
		}
		resolvedProtos = append(resolvedProtos, b)
	}
	for _, p := range importPaths {
		d, _ := filepath.Split(p)
		if d != prefix {
			return nil, nil, nil, errors.New("could not resolve paths")
		}
	}
	for _, p := range protos {
		d, _ := filepath.Split(p)
		if d != prefix {
			return nil, nil, nil, errors.New("could not resolve paths")
		}
	}
	return resolvedIPaths, resolvedProtos, func(filename string) (io.ReadCloser, error) {
		return os.Open(filepath.Join(prefix, filename))
	}, nil
}

func serviceMatchFunc(service string) matchFunc {
	return func(r *Request) bool {
		return r.Service == strings.TrimPrefix(service, "/")
	}
}

func methodMatchFunc(method string) matchFunc {
	return func(r *Request) bool {
		if !strings.Contains(method, "/") {
			return r.Method == method
		}
		splitted := strings.Split(strings.TrimPrefix(method, "/"), "/")
		s := strings.Join(splitted[:len(splitted)-1], "/")
		m := splitted[len(splitted)-1]
		return r.Service == s && r.Method == m
	}
}

func registerFileDescriptors(fds []*desc.FileDescriptor) (err error) {
	var registry *protoregistry.Files
	registry, err = protodesc.NewFiles(desc.ToFileDescriptorSet(fds...))
	if err != nil {
		return err
	}
	registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if ofd, _ := protoregistry.GlobalFiles.FindFileByPath(fd.Path()); ofd != nil {
			return true
		}
		err = protoregistry.GlobalFiles.RegisterFile(fd)
		return (err == nil)
	})
	return
}
