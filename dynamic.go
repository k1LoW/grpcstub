package grpcstub

import (
	"math/rand"
	"strings"

	"github.com/jaswdr/faker"
	"github.com/jhump/protoreflect/desc"
	"github.com/minio/pkg/wildcard"
	"google.golang.org/protobuf/types/descriptorpb"
)

var fk = faker.New()

type generator struct {
	pattern string
	fn      GenerateFunc
}

type generators []*generator

func (gs generators) matchFunc(name string) (GenerateFunc, bool) {
	for _, g := range gs {
		if wildcard.MatchSimple(g.pattern, name) {
			return g.fn, true
		}
	}
	return nil, false
}

type GeneratorOption func(generators) generators

type GenerateFunc func(r *Request) interface{}

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
	m.handler = func(r *Request, md *desc.MethodDescriptor) *Response {
		var res *Response
		if prev == nil {
			res = NewResponse()
		} else {
			res = prev(r, md)
		}
		if !md.IsClientStreaming() && !md.IsServerStreaming() {
			res.Messages = append(res.Messages, generateDynamicMessage(gs, r, md.GetOutputType(), nil))
		} else {
			for i := 0; i > rand.Intn(messageMax)+1; i++ {
				res.Messages = append(res.Messages, generateDynamicMessage(gs, r, md.GetOutputType(), nil))
			}
		}
		return res
	}
	return m
}

func generateDynamicMessage(gs generators, r *Request, m *desc.MessageDescriptor, parents []string) map[string]interface{} {
	const (
		floatMin  = 0
		floatMax  = 10000
		wMin      = 1
		wMax      = 25
		repeatMax = 5
		fieldSep  = "."
	)
	message := map[string]interface{}{}
	for _, f := range m.GetFields() {
		values := []interface{}{}
		l := 1
		if f.IsProto3Optional() {
			l = rand.Intn(2)
		}
		if f.IsRepeated() {
			l = rand.Intn(repeatMax) + l
		}
		n := f.GetName()
		names := append(parents, n)
		for i := 0; i < l; i++ {
			fn, ok := gs.matchFunc(strings.Join(names, fieldSep))
			if ok {
				values = append(values, cast(fn(r)))
				continue
			}
			switch f.GetType() {
			case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
				values = append(values, fk.Float64(1, floatMin, floatMax))
			case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, descriptorpb.FieldDescriptorProto_TYPE_SINT64:
				values = append(values, fk.Int64())
			case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_SINT32:
				values = append(values, fk.Int32())
			case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
				values = append(values, fk.UInt64())
			case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
				values = append(values, fk.UInt32())
			case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
				values = append(values, fk.Bool())
			case descriptorpb.FieldDescriptorProto_TYPE_STRING:
				values = append(values, fk.Lorem().Sentence(rand.Intn(wMax-wMin+1)+wMin))
			case descriptorpb.FieldDescriptorProto_TYPE_GROUP:
				// Group type is deprecated and not supported in proto3.
			case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
				values = append(values, generateDynamicMessage(gs, r, f.GetMessageType(), names))
			case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
				values = append(values, fk.Lorem().Bytes(rand.Intn(wMax-wMin+1)+wMin))
			case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
				values = append(values, f.GetEnumType().GetValues()[0].GetNumber())
			}
		}
		if f.IsRepeated() {
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
