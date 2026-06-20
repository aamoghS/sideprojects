package server

import (
	"context"
	"io"
	"net"

	relayv1 "relay/gen/relay/v1"
	"relay/internal/store"

	"google.golang.org/grpc"
)

type Server struct {
	relayv1.UnimplementedRelayServiceServer
	events *store.Store
	rooms  *store.RoomStore
}

func New(events *store.Store, rooms *store.RoomStore) *Server {
	return &Server{events: events, rooms: rooms}
}

func ListenAndServe(addr string, srv *Server) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	grpcSrv := grpc.NewServer()
	relayv1.RegisterRelayServiceServer(grpcSrv, srv)
	return grpcSrv.Serve(lis)
}

func (s *Server) Publish(_ context.Context, req *relayv1.PublishRequest) (*relayv1.PublishResponse, error) {
	ev, count := s.events.Publish(req.GetTopic(), req.GetPayload(), req.GetMetadata())
	return &relayv1.PublishResponse{
		EventId:          ev.ID,
		SubscriberCount:  int32(count),
		Offset:           ev.Offset,
	}, nil
}

func (s *Server) Subscribe(req *relayv1.SubscribeRequest, stream relayv1.RelayService_SubscribeServer) error {
	ch, unsub := s.events.Subscribe(req.GetTopics(), req.GetFromOffset())
	defer unsub()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(toProtoEvent(ev)); err != nil {
				return err
			}
		}
	}
}

func (s *Server) JoinRoom(stream relayv1.RelayService_JoinRoomServer) error {
	var (
		room     string
		nickname string
		leave    func()
	)

	defer func() {
		if leave != nil {
			leave()
		}
	}()

	for {
		frame, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch body := frame.GetBody().(type) {
		case *relayv1.RoomFrame_Join:
			if room != "" {
				continue
			}
			room = body.Join.GetRoom()
			nickname = body.Join.GetNickname()
			if nickname == "" {
				nickname = "anon"
			}
			ch, l := s.rooms.Join(room, nickname)
			leave = l
			go func() {
				for f := range ch {
					if err := stream.Send(toProtoRoomFrame(f)); err != nil {
						return
					}
				}
			}()
		case *relayv1.RoomFrame_Chat:
			if room == "" {
				continue
			}
			s.rooms.Chat(room, nickname, body.Chat.GetText())
		case *relayv1.RoomFrame_Leave:
			return nil
		}
	}
}

func toProtoEvent(ev store.Event) *relayv1.Event {
	return &relayv1.Event{
		Id:                ev.ID,
		Topic:             ev.Topic,
		Payload:           ev.Payload,
		TimestampUnixNano: ev.Timestamp.UnixNano(),
		Metadata:          ev.Metadata,
		Offset:            ev.Offset,
	}
}

func toProtoRoomFrame(frame store.RoomFrame) *relayv1.RoomFrame {
	if frame.Chat != nil {
		return &relayv1.RoomFrame{
			Body: &relayv1.RoomFrame_Chat{
				Chat: &relayv1.RoomChat{
					Room:                frame.Chat.Room,
					Sender:              frame.Chat.Sender,
					Text:                frame.Chat.Text,
					TimestampUnixNano: frame.Chat.Timestamp.UnixNano(),
				},
			},
		}
	}
	p := frame.Presence
	kind := relayv1.PresenceKind_PRESENCE_KIND_UNSPECIFIED
	switch p.Kind {
	case store.PresenceJoin:
		kind = relayv1.PresenceKind_PRESENCE_KIND_JOIN
	case store.PresenceLeave:
		kind = relayv1.PresenceKind_PRESENCE_KIND_LEAVE
	}
	return &relayv1.RoomFrame{
		Body: &relayv1.RoomFrame_Presence{
			Presence: &relayv1.RoomPresence{
				Room:             p.Room,
				Kind:             kind,
				Participant:      p.Participant,
				ParticipantCount: int32(p.ParticipantCount),
			},
		},
	}
}
