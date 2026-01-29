package reply

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/line/line-bot-sdk-go/v7/linebot"

	"github.com/94peter/botreplyer/provider/line/reply/textreply"
)

type Reply interface {
	WelcomeReply(ctx context.Context, userID string) ([]linebot.SendingMessage, error)
	MessageTextReply(
		ctx context.Context, typ linebot.EventSourceType,
		groupId, userID, msg string, session sessions.Session,
	) ([]linebot.SendingMessage, textreply.DelayedMessage, error)
	JoinGroupReply(ctx context.Context) ([]linebot.SendingMessage, error)
}
