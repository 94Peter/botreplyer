package line

import (
	"github.com/94peter/botreplyer/follow"
	"github.com/94peter/botreplyer/group"
	"github.com/94peter/botreplyer/provider/line/reply"
	"github.com/94peter/botreplyer/provider/line/reply/textreply"
)

type Config struct {
	ChannelSecret string
	ChannelToken  string
	AdminUserId   string

	Replies []textreply.LineKeywordReply

	FollowStore follow.Store
	GroupStore  group.Store

	JoinGroupReplyFunc reply.JoinGroupReplyFunc
}

var DefaultConfig = &Config{}

type Option func(*Config)

func WithChannelSecret(secret string) Option {
	return func(c *Config) {
		c.ChannelSecret = secret
	}
}

func WithChannelToken(token string) Option {
	return func(c *Config) {
		c.ChannelToken = token
	}
}

func WithReplies(replies ...textreply.LineKeywordReply) Option {
	return func(c *Config) {
		c.Replies = replies
	}
}

func WithAdminUserId(userId string) Option {
	return func(c *Config) {
		c.AdminUserId = userId
	}
}

func WithJoinGroupReplyFunc(f reply.JoinGroupReplyFunc) Option {
	return func(c *Config) {
		c.JoinGroupReplyFunc = f
	}
}
