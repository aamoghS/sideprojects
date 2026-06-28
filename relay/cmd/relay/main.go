package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	relayv1 "github.com/aamoghS/sideprojects/relay/gen/relay/v1"
	"github.com/aamoghS/sideprojects/relay/internal/client"
	"github.com/aamoghS/sideprojects/relay/internal/server"
	"github.com/aamoghS/sideprojects/relay/internal/store"
)

const defaultAddr = ":50051"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
	case "publish":
		cmdPublish(os.Args[2:])
	case "watch":
		cmdWatch(os.Args[2:])
	case "room":
		cmdRoom(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `relay - real-time event bus over gRPC

Usage:
  relay serve [--addr :50051]
  relay publish <topic> <message> [--addr :50051] [--meta key=val]
  relay watch <topic>... [--addr :50051] [--replay N]
  relay room <room-name> [--addr :50051] [--name nickname]

`)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", defaultAddr, "listen address")
	_ = fs.Parse(args)

	srv := server.New(store.New(), store.NewRoomStore())
	fmt.Fprintf(os.Stderr, "relay listening on %s\n", *addr)
	if err := server.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func cmdPublish(args []string) {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	addr := fs.String("addr", defaultAddr, "relay server address")
	meta := fs.String("meta", "", "metadata as key=val pairs separated by commas")
	flagArgs, positional := splitFlags(args)
	_ = fs.Parse(flagArgs)

	if len(positional) < 2 {
		fmt.Fprintln(os.Stderr, "usage: relay publish <topic> <message>")
		os.Exit(1)
	}

	topic := positional[0]
	message := strings.Join(positional[1:], " ")
	metadata := parseMeta(*meta)

	c, err := client.Dial(*addr)
	if err != nil {
		fail(err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.Publish(ctx, topic, message, metadata)
	if err != nil {
		fail(err)
	}
	fmt.Printf("published %s to %s (%d subscribers, offset %d)\n",
		resp.GetEventId(), topic, resp.GetSubscriberCount(), resp.GetOffset())
}

func cmdWatch(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	addr := fs.String("addr", defaultAddr, "relay server address")
	replay := fs.Int64("replay", 0, "replay events from offset")
	flagArgs, positional := splitFlags(args)
	_ = fs.Parse(flagArgs)

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "usage: relay watch <topic>...")
		os.Exit(1)
	}
	topics := positional
	colors := make(map[string]func(string) string, len(topics))
	for _, t := range topics {
		colors[t] = client.TopicColor(t)
	}

	c, err := client.Dial(*addr)
	if err != nil {
		fail(err)
	}
	defer c.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err = c.Watch(ctx, topics, *replay, func(ev *relayv1.Event) {
		colorFn := colors[ev.GetTopic()]
		if colorFn == nil {
			colorFn = func(s string) string { return s }
		}
		fmt.Println(client.FormatEvent(ev, colorFn))
	})
	if err != nil && ctx.Err() == nil {
		fail(err)
	}
}

func cmdRoom(args []string) {
	fs := flag.NewFlagSet("room", flag.ExitOnError)
	addr := fs.String("addr", defaultAddr, "relay server address")
	name := fs.String("name", os.Getenv("USER"), "nickname in room")
	flagArgs, positional := splitFlags(args)
	_ = fs.Parse(flagArgs)

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "usage: relay room <room-name>")
		os.Exit(1)
	}
	room := positional[0]
	if *name == "" {
		*name = "anon"
	}

	c, err := client.Dial(*addr)
	if err != nil {
		fail(err)
	}
	defer c.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	session, err := c.JoinRoom(ctx, room, *name)
	if err != nil {
		fail(err)
	}
	defer session.Leave()

	fmt.Fprintf(os.Stderr, "joined room %s as %s (/quit to leave)\n", room, *name)

	go func() {
		for {
			frame, err := session.Recv()
			if err != nil {
				return
			}
			if frame.GetPresence() != nil {
				client.PrintPresence(frame)
				continue
			}
			client.PrintChat(frame)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "/quit" {
			break
		}
		if err := session.SendChat(line); err != nil {
			fail(err)
		}
	}
}

func parseMeta(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		k, v, ok := strings.Cut(part, "=")
		if !ok || k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func splitFlags(args []string) (flags []string, positional []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return flags, append(positional, args[i+1:]...)
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.Contains(arg, "=") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, arg)
	}
	return flags, positional
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
