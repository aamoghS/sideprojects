// grokbot: tiny Discord CLI + MCP server for Grok Bot.
//
// Usage:
//
//	grokbot guilds
//	grokbot channels <guild_id>
//	grokbot read <channel_id> [limit]
//	grokbot send <channel_id> "message"
//	grokbot say "message"  (webhook, posts as you)
//	grokbot mcp            (run as stdio MCP server)
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const api = "https://discord.com/api/v10"

const usage = `usage:
  grokbot guilds
  grokbot channels <guild_id>
  grokbot read <channel_id> [limit]
  grokbot send <channel_id> "message"
  grokbot say "message"      (webhook, posts as you)
  grokbot mcp`

type Item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	Author  string `json:"author"`
	Content string `json:"content"`
}

// loadEnv reads key from the environment, else from .env in the cwd or next to the binary.
func loadEnv(key string) (string, error) {
	if t := os.Getenv(key); t != "" {
		return t, nil
	}
	exe, _ := os.Executable()
	for _, dir := range []string{".", filepath.Dir(exe)} {
		f, err := os.Open(filepath.Join(dir, ".env"))
		if err != nil {
			continue
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		for s.Scan() {
			if v, ok := strings.CutPrefix(s.Text(), key+"="); ok {
				return strings.TrimSpace(v), nil
			}
		}
	}
	return "", fmt.Errorf("%s not set (env var or .env file)", key)
}

func call(method, path string, body, out any) error {
	token, err := loadEnv("DISCORD_TOKEN")
	if err != nil {
		return err
	}
	return do(method, api+path, "Bot "+token, body, out)
}

func do(method, url, auth string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rd)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DiscordBot (grokbot, 1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord %s: %s", resp.Status, data)
	}
	return json.Unmarshal(data, out)
}

func listGuilds() ([]Item, error) {
	var gs []Item
	err := call("GET", "/users/@me/guilds", nil, &gs)
	return gs, err
}

func listChannels(guildID string) ([]Item, error) {
	var cs []struct {
		Item
		Type int `json:"type"`
	}
	if err := call("GET", "/guilds/"+guildID+"/channels", nil, &cs); err != nil {
		return nil, err
	}
	out := []Item{}
	for _, c := range cs {
		if c.Type == 0 {
			out = append(out, c.Item)
		}
	}
	return out, nil
}

func readMessages(channelID string, limit int) ([]Message, error) {
	var ms []struct {
		Author  struct{ Username string } `json:"author"`
		Content string                    `json:"content"`
	}
	if err := call("GET", fmt.Sprintf("/channels/%s/messages?limit=%d", channelID, limit), nil, &ms); err != nil {
		return nil, err
	}
	out := make([]Message, len(ms))
	for i, m := range ms { // Discord returns newest first; flip to oldest first
		out[len(ms)-1-i] = Message{m.Author.Username, m.Content}
	}
	return out, nil
}

func sendMessage(channelID, content string) (string, error) {
	var m struct{ ID string }
	err := call("POST", "/channels/"+channelID+"/messages", map[string]string{"content": content}, &m)
	return m.ID, err
}

// sayAsMe posts via the webhook in DISCORD_WEBHOOK_URL, which shows the
// webhook's name and avatar (set them to yours in Discord).
func sayAsMe(content string) (string, error) {
	url, err := loadEnv("DISCORD_WEBHOOK_URL")
	if err != nil {
		return "", err
	}
	var m struct{ ID string }
	err = do("POST", url+"?wait=true", "", map[string]string{"content": content}, &m)
	return m.ID, err
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fail(usage)
	}
	var out any
	var err error
	switch {
	case args[0] == "mcp":
		err = runMCP()
		if err != nil {
			fail(err.Error())
		}
		return
	case args[0] == "guilds":
		out, err = listGuilds()
	case args[0] == "channels" && len(args) == 2:
		out, err = listChannels(args[1])
	case args[0] == "read" && (len(args) == 2 || len(args) == 3):
		limit := 20
		if len(args) == 3 {
			if limit, err = strconv.Atoi(args[2]); err != nil {
				fail("limit must be a number")
			}
		}
		out, err = readMessages(args[1], limit)
	case args[0] == "send" && len(args) == 3:
		var id string
		id, err = sendMessage(args[1], args[2])
		out = map[string]string{"sent_id": id}
	case args[0] == "say" && len(args) == 2:
		var id string
		id, err = sayAsMe(args[1])
		out = map[string]string{"sent_id": id}
	default:
		fail(usage)
	}
	if err != nil {
		fail(err.Error())
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// --- MCP server ---

type guildIn struct {
	GuildID string `json:"guild_id" jsonschema:"guild (server) id"`
}

type readIn struct {
	ChannelID string `json:"channel_id" jsonschema:"channel id"`
	Limit     int    `json:"limit,omitempty" jsonschema:"number of messages, default 20"`
}

type sayIn struct {
	Content string `json:"content" jsonschema:"message text"`
}

type sendIn struct {
	ChannelID string `json:"channel_id" jsonschema:"channel id"`
	Content   string `json:"content" jsonschema:"message text"`
}

// text wraps any value as a JSON text result.
func text(v any, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, err
	}
	b, _ := json.Marshal(v)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func runMCP() error {
	s := mcp.NewServer(&mcp.Implementation{Name: "discord", Version: "1.0.0"}, nil)

	mcp.AddTool(s, &mcp.Tool{Name: "list_guilds", Description: "List servers (guilds) the bot is in."},
		func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			return text(listGuilds())
		})
	mcp.AddTool(s, &mcp.Tool{Name: "list_channels", Description: "List text channels in a guild."},
		func(ctx context.Context, req *mcp.CallToolRequest, in guildIn) (*mcp.CallToolResult, any, error) {
			return text(listChannels(in.GuildID))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "read_messages", Description: "Read recent messages from a channel, oldest first."},
		func(ctx context.Context, req *mcp.CallToolRequest, in readIn) (*mcp.CallToolResult, any, error) {
			if in.Limit == 0 {
				in.Limit = 20
			}
			return text(readMessages(in.ChannelID, in.Limit))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "send_message", Description: "Send a message to a channel. Returns the message id."},
		func(ctx context.Context, req *mcp.CallToolRequest, in sendIn) (*mcp.CallToolResult, any, error) {
			id, err := sendMessage(in.ChannelID, in.Content)
			return text(map[string]string{"sent_id": id}, err)
		})

	mcp.AddTool(s, &mcp.Tool{Name: "send_as_me", Description: "Send a message as the user (via webhook, shows the user's name and avatar) to the webhook's channel."},
		func(ctx context.Context, req *mcp.CallToolRequest, in sayIn) (*mcp.CallToolResult, any, error) {
			id, err := sayAsMe(in.Content)
			return text(map[string]string{"sent_id": id}, err)
		})

	return s.Run(context.Background(), &mcp.StdioTransport{})
}
