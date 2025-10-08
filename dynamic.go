package grpcstub

import (
	"math/rand"
	"strings"
	"time"

	wildcard "github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/jaswdr/faker"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var fk = faker.New()

type generator struct {
	pattern string
	fn      GenerateFunc
}

type generators []*generator

func (gs generators) matchFunc(name string) (GenerateFunc, bool) {
	for _, g := range gs {
		if wildcard.Match(g.pattern, name) {
			return g.fn, true
		}
	}
	return nil, false
}

type GeneratorOption func(generators) generators

type GenerateFunc func(req *Request) any

func Generator(pattern string, fn GenerateFunc) GeneratorOption {
	return func(gs generators) generators {
		return append(gs, &generator{
			pattern: pattern,
			fn:      fn,
		})
	}
}

// ResponseDynamic set handler which return dynamic response.
func (m *matcher) ResponseDynamic(opts ...GeneratorOption) *matcher {
	const messageMax = 5
	gs := generators{}
	for _, opt := range opts {
		gs = opt(gs)
	}
	prev := m.handler
	m.handler = func(req *Request, md protoreflect.MethodDescriptor) *Response {
		var res *Response
		if prev == nil {
			res = NewResponse()
		} else {
			res = prev(req, md)
		}
		if !md.IsStreamingClient() && !md.IsStreamingServer() {
			res.Messages = append(res.Messages, generateDynamicMessage(gs, req, md.Output(), nil))
		} else {
			for i := 0; i > rand.Intn(messageMax)+1; i++ {
				res.Messages = append(res.Messages, generateDynamicMessage(gs, req, md.Output(), nil))
			}
		}
		return res
	}
	return m
}

func generateDynamicMessage(gs generators, req *Request, m protoreflect.MessageDescriptor, parents []string) map[string]any {
	const (
		floatMin  = 0
		floatMax  = 10000
		wMin      = 1
		wMax      = 25
		repeatMax = 5
		fieldSep  = "."
	)
	message := map[string]any{}

	for i := 0; i < m.Fields().Len(); i++ {
		f := m.Fields().Get(i)
		values := []any{}
		l := 1
		if f.HasOptionalKeyword() {
			l = rand.Intn(2)
		}
		if f.IsList() {
			l = rand.Intn(repeatMax) + l
		}
		n := string(f.Name())
		names := append(parents, string(n))
		for i := 0; i < l; i++ {
			fn, ok := gs.matchFunc(strings.Join(names, fieldSep))
			if ok {
				values = append(values, fn(req))
				continue
			}
			switch f.Kind() {
			case protoreflect.DoubleKind, protoreflect.FloatKind:
				values = append(values, fk.Float64(1, floatMin, floatMax))
			case protoreflect.Int64Kind, protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind, protoreflect.Sint64Kind:
				values = append(values, fk.Int64())
			case protoreflect.Int32Kind, protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind:
				values = append(values, fk.Int32())
			case protoreflect.Uint64Kind:
				values = append(values, fk.UInt64())
			case protoreflect.Uint32Kind:
				values = append(values, fk.UInt32())
			case protoreflect.BoolKind:
				values = append(values, fk.Bool())
			case protoreflect.StringKind:
				values = append(values, fk.Lorem().Sentence(rand.Intn(wMax-wMin+1)+wMin))
			case protoreflect.GroupKind:
				// Group type is deprecated and not supported in proto3.
			case protoreflect.MessageKind:
				if f.Message().FullName() == "google.protobuf.Timestamp" {
					// Timestamp is not encoded as a message with seconds and nanos in JSON, instead it is encoded with RFC 3339:
					// ref: https://protobuf.dev/programming-guides/proto3/#json
					values = append(values, fk.Time().Time(time.Now()).Format(time.RFC3339Nano))
					continue
				}
				values = append(values, generateDynamicMessage(gs, req, f.Message(), names))
			case protoreflect.BytesKind:
				values = append(values, fk.Lorem().Bytes(rand.Intn(wMax-wMin+1)+wMin))
			case protoreflect.EnumKind:
				values = append(values, int(f.Enum().Values().Get(0).Number()))
			}
		}
		if f.IsList() {
			message[n] = values
		} else {
			if len(values) > 0 {
				message[n] = values[0]
			}
		}
	}

	return message
}

// ResponseDynamic set handler which return dynamic response.
func (s *Server) ResponseDynamic(opts ...GeneratorOption) *matcher {
	m := &matcher{
		matchFuncs: []matchFunc{func(_ *Request) bool { return true }},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matchers = append(s.matchers, m)
	return m.ResponseDynamic(opts...)
}
