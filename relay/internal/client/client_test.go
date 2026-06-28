package client

import (
	"context"
	"net"
	"testing"
	"time"

	relayv1 "github.com/aamoghS/sideprojects/relay/gen/relay/v1"
	"github.com/aamoghS/sideprojects/relay/internal/server"
	"github.com/aamoghS/sideprojects/relay/internal/store"

	"google.golang.org/grpc"
)

func TestClientPublish(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	gs := grpc.NewServer()
	relayv1.RegisterRelayServiceServer(gs, server.New(store.New(), store.NewRoomStore()))
	go gs.Serve(lis)
	defer gs.Stop()

	c, err := Dial(lis.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := c.Publish(ctx, "dev", "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetEventId() == "" {
		t.Fatal("missing event id")
	}
}
