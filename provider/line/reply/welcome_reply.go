package reply

import (
	"context"

	"github.com/line/line-bot-sdk-go/v7/linebot"

	"github.com/94peter/botreplyer/provider/line/flexmsg"
)

func (svc *replyImpl) WelcomeReply(ctx context.Context, userID string) ([]linebot.SendingMessage, error) {
	flex := flexmsg.WelcomeFlex{}
	msg, err := flex.Build()
	if err != nil {
		return nil, err
	}
	return []linebot.SendingMessage{
		msg,
	}, nil
}
