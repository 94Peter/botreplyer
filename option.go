package botreplyer

import (
	"github.com/94peter/botreplyer/provider/line"
	"github.com/94peter/botreplyer/provider/line/reply"
)

type Option func(*config) error

func WithLineConfig(opts ...line.Option) Option {
	return func(c *config) error {
		cfg := line.DefaultConfig
		for _, opt := range opts {
			opt(cfg)
		}
		c.lineConfig = cfg
		return nil
	}
}

func WithJoinGroupReplyFunc(f reply.JoinGroupReplyFunc) Option {
	return func(c *config) error {
		c.lineConfig.JoinGroupReplyFunc = f
		return nil
	}
}
