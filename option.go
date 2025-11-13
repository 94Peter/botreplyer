package botreplyer

import (
	"github.com/94peter/botreplyer/provider/line"
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
