package grpcstub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/k1LoW/grpcstub/testdata/routeguide"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBidiStreaming(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RouteChat").Match(func(req *Request) bool {
		m, ok := req.Message["message"]
		if !ok {
			return false
		}
		return strings.Contains(m.(string), "hello from client[0]")
	}).Header("hello", "header").
		Response(map[string]any{"location": nil, "message": "hello from server[0]"})
	ts.Method("RouteChat").
		Header("hello", "header").
		Handler(func(req *Request) *Response {
			res := NewResponse()
			m, ok := req.Message["message"]
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

func TestBidiStreamingUnmatched(t *testing.T) {
	ctx := context.Background()
	ts := NewServer(t, "testdata/route_guide.proto")
	t.Cleanup(func() {
		ts.Close()
	})
	ts.Method("RouteChat").Match(func(req *Request) bool {
		return false
	}).Header("hello", "header").
		Response(map[string]any{"location": nil, "message": "hello from server[0]"})

	client := routeguide.NewRouteGuideClient(ts.Conn())
	stream, err := client.RouteChat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.SendMsg(&routeguide.RouteNote{
		Message: fmt.Sprintf("hello from client[%d]", 0),
	}); err != nil {
		t.Error("want error")
	}
	if _, err := stream.Recv(); err == nil {
		t.Error("want error")
	}

	{
		got := len(ts.Requests())
		if want := 0; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}

	{
		got := len(ts.UnmatchedRequests())
		if want := 1; got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	}
}
