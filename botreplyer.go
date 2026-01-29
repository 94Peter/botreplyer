package botreplyer

import (
	"context"

	"github.com/94peter/botreplyer/follow"
	followMongo "github.com/94peter/botreplyer/follow/mongo"
	groupMongo "github.com/94peter/botreplyer/group/mongo"
	"github.com/94peter/botreplyer/handler"
	"github.com/94peter/botreplyer/provider/line"
	"github.com/94peter/botreplyer/provider/line/reply"

	"go.opentelemetry.io/otel"
)

type config struct {
	lineConfig *line.Config
}

func InitBotReplyer(ctx context.Context, options ...Option) error {
	// init bot replyer
	c := &config{}
	for _, option := range options {
		err := option(c)
		if err != nil {
			return err
		}
	}
	var err error
	// sessionStore, err := sessionMongo.NewSessionStore(ctx)
	// if err != nil {
	// 	return err
	// }
	followStore, err := followMongo.NewStore(ctx)
	if err != nil {
		return err
	}
	groupStore, err := groupMongo.NewStore(ctx)
	if err != nil {
		return err
	}
	if c.lineConfig != nil {
		// init line bot
		// c.lineConfig.SessionStore = sessionStore
		c.lineConfig.FollowStore = followStore
		c.lineConfig.GroupStore = groupStore
		err := initLineBot(c.lineConfig)
		if err != nil {
			return err
		}
	}
	_FollowStore = followStore
	return nil
}

var _FollowStore follow.Store

func GetFollowStore() follow.Store {
	if _FollowStore == nil {
		panic("follow store is nil, please init bot replyer first")
	}
	return _FollowStore
}

func initLineBot(cfg *line.Config) error {
	sdk, err := line.NewSDK(cfg.ChannelSecret, cfg.ChannelToken)
	if err != nil {
		return err
	}
	replyTracer := otel.Tracer("BotReplier")
	handler.InitLinebotWebhook(
		handler.WithLineSDK(sdk),
		handler.WithLineMsgReplyService(
			reply.NewReply(
				reply.WithTextReply(cfg.Replies...),
				// reply.WithStore(cfg.SessionStore),
				reply.WithJoinGroupReply(cfg.JoinGroupReplyFunc),
				reply.WithTracer(replyTracer),
			),
		),
		handler.WithFollowStore(cfg.FollowStore),
		handler.WithGroupStore(cfg.GroupStore),
		handler.WithAdminUserId(cfg.AdminUserId),
		handler.WithLineNotificationService(cfg.NotificationService),
	)
	return nil
}
