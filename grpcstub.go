package grpcstub

import (
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/golang/protobuf/proto" //nolint
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mattn/go-jsonpointer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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

func (s *Server) Close() {
	s.t.Helper()
	if s.listener == nil {
		s.t.Error("server is not started yet")
		return
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

func (s *Server) Conn() *grpc.ClientConn {
	s.t.Helper()
	if s.listener == nil {
		s.t.Error("server is not started yet")
		return nil
	}
	conn, err := grpc.Dial(s.listener.Addr().String(), grpc.WithInsecure())
	if err != nil {
		s.t.Error(err)
		return nil
	}
	return conn
}

func (s *Server) startServer() {
	s.t.Helper()
	s.registerServer()
	reflection.Register(s.server)
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

func (s *Server) Match(fn func(r *Request) bool) *matcher {
	m := &matcher{
		matchFuncs: []matchFunc{fn},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matchers = append(s.matchers, m)
	return m
}

func (m *matcher) Match(fn func(r *Request) bool) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

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

func (m *matcher) Service(service string) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn := serviceMatchFunc(service)
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

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

func (m *matcher) Method(method string) *matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn := methodMatchFunc(method)
	m.matchFuncs = append(m.matchFuncs, fn)
	return m
}

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

func (m *matcher) Handler(fn func(r *Request) *Response) {
	m.handler = fn
}

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

func (m *matcher) ResponseString(message string) *matcher {
	mes := make(map[string]interface{})
	_ = json.Unmarshal([]byte(message), &mes)
	return m.Response(mes)
}

func (s *Server) Requests() []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requests
}

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
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		m := Message{}
		if err := json.Unmarshal(b, &m); err != nil {
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
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		m := Message{}
		if err := json.Unmarshal(b, &m); err != nil {
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
				b, err := json.Marshal(in)
				if err != nil {
					return err
				}
				m := Message{}
				if err := json.Unmarshal(b, &m); err != nil {
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
						}
						return stream.SendMsg(mes)
					}
					return status.Error(codes.NotFound, codes.NotFound.String())
				}
			}
			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) createBiStreamingHandler(md *desc.MethodDescriptor) func(srv interface{}, stream grpc.ServerStream) error {
	return func(srv interface{}, stream grpc.ServerStream) error {
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
			b, err := json.Marshal(in)
			if err != nil {
				return err
			}
			m := Message{}
			if err := json.Unmarshal(b, &m); err != nil {
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
	p := protoparse.Parser{
		ImportPaths:           importPaths,
		InferImportPaths:      len(importPaths) == 0,
		IncludeSourceCodeInfo: true,
	}
	fds, err := p.ParseFiles(protos...)
	if err != nil {
		return nil, err
	}
	return fds, nil
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
