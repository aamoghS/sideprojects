package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	relayv1 "relay/gen/relay/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	api    relayv1.RelayServiceClient
	addr   string
}

func Dial(addr string) (*Client, error) {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
		api:  relayv1.NewRelayServiceClient(conn),
		addr: addr,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Publish(ctx context.Context, topic, message string, metadata map[string]string) (*relayv1.PublishResponse, error) {
	return c.api.Publish(ctx, &relayv1.PublishRequest{
		Topic:    topic,
		Payload:  []byte(message),
		Metadata: metadata,
	})
}

func (c *Client) Watch(ctx context.Context, topics []string, fromOffset int64, onEvent func(*relayv1.Event)) error {
	stream, err := c.api.Subscribe(ctx, &relayv1.SubscribeRequest{
		Topics:     topics,
		FromOffset: fromOffset,
	})
	if err != nil {
		return err
	}
	for {
		ev, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		onEvent(ev)
	}
}

type RoomSession struct {
	stream relayv1.RelayService_JoinRoomClient
	room   string
	name   string
}

func (c *Client) JoinRoom(ctx context.Context, room, nickname string) (*RoomSession, error) {
	stream, err := c.api.JoinRoom(ctx)
	if err != nil {
		return nil, err
	}
	s := &RoomSession{stream: stream, room: room, name: nickname}
	if err := stream.Send(&relayv1.RoomFrame{
		Body: &relayv1.RoomFrame_Join{
			Join: &relayv1.RoomJoin{Room: room, Nickname: nickname},
		},
	}); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *RoomSession) SendChat(text string) error {
	return s.stream.Send(&relayv1.RoomFrame{
		Body: &relayv1.RoomFrame_Chat{
			Chat: &relayv1.RoomChat{
				Room: s.room,
				Text: text,
			},
		},
	})
}

func (s *RoomSession) Recv() (*relayv1.RoomFrame, error) {
	return s.stream.Recv()
}

func (s *RoomSession) Leave() error {
	_ = s.stream.Send(&relayv1.RoomFrame{
		Body: &relayv1.RoomFrame_Leave{Leave: &relayv1.RoomLeave{Room: s.room}},
	})
	return s.stream.CloseSend()
}

func FormatEvent(ev interface {
	GetTopic() string
	GetTimestampUnixNano() int64
	GetOffset() int64
	GetPayload() []byte
	GetMetadata() map[string]string
}, topicColor func(string) string) string {
	ts := time.Unix(0, ev.GetTimestampUnixNano()).UTC().Format("15:04:05.000")
	topic := topicColor(ev.GetTopic())
	meta := ""
	if len(ev.GetMetadata()) > 0 {
		meta = fmt.Sprintf(" %v", ev.GetMetadata())
	}
	return fmt.Sprintf("%s %s #%d %s%s", ts, topic, ev.GetOffset(), string(ev.GetPayload()), meta)
}

func TopicColor(topic string) func(string) string {
	colors := []func(a ...interface{}) string{
		func(a ...interface{}) string { return fmt.Sprint(a...) },
	}
	_ = colors
	palette := []string{"\033[36m", "\033[33m", "\033[35m", "\033[32m", "\033[34m"}
	reset := "\033[0m"
	idx := 0
	for _, ch := range topic {
		idx += int(ch)
	}
	color := palette[idx%len(palette)]
	return func(t string) string {
		return color + t + reset
	}
}

func PrintPresence(frame *relayv1.RoomFrame) {
	p := frame.GetPresence()
	if p == nil {
		return
	}
	switch p.GetKind() {
	case relayv1.PresenceKind_PRESENCE_KIND_JOIN:
		fmt.Fprintf(os.Stderr, "* %s joined (%d here)\n", p.GetParticipant(), p.GetParticipantCount())
	case relayv1.PresenceKind_PRESENCE_KIND_LEAVE:
		fmt.Fprintf(os.Stderr, "* %s left (%d here)\n", p.GetParticipant(), p.GetParticipantCount())
	}
}

func PrintChat(frame *relayv1.RoomFrame) {
	c := frame.GetChat()
	if c == nil {
		return
	}
	ts := time.Unix(0, c.GetTimestampUnixNano()).UTC().Format("15:04:05")
	fmt.Printf("[%s] %s: %s\n", ts, c.GetSender(), c.GetText())
}
