package textreply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-contrib/sessions"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"gopkg.in/yaml.v3"
)

type QuickReply struct {
	Label string `yaml:"label"`
	Text  string `yaml:"text"`
}

var cfgRoot map[string]yaml.Node

func Load(path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to read message file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfgRoot); err != nil {
		panic(err)
	}
	return nil
}

func GetNode(key string) (yaml.Node, bool) {
	data, ok := cfgRoot[key]
	return data, ok
}

type SessionHandler func(session sessions.Session)

type DelayedMessage chan []linebot.SendingMessage

type LineKeywordReply interface {
	MessageTextReply(
		ctx context.Context,
		typ linebot.EventSourceType,
		groupID, userID, msg string,
		session sessions.Session,
	) ([]linebot.SendingMessage, DelayedMessage, error)
}
