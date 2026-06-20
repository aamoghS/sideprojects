package server

import (
	"context"
	"net"
	"testing"

	relayv1 "relay/gen/relay/v1"
	"relay/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPublishRoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	gs := grpc.NewServer()
	relayv1.RegisterRelayServiceServer(gs, New(store.New(), store.NewRoomStore()))
	go gs.Serve(lis)
	defer gs.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := relayv1.NewRelayServiceClient(conn)
	resp, err := client.Publish(context.Background(), &relayv1.PublishRequest{
		Topic:   "dev",
		Payload: []byte("ping"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetEventId() == "" {
		t.Fatal("expected event id")
	}
}
