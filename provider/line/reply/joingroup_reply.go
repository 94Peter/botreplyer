package reply

import (
	"context"

	"github.com/line/line-bot-sdk-go/v7/linebot"

	"github.com/94peter/botreplyer/provider/line/flexmsg"
)

type JoinGroupReplyFunc func(ctx context.Context) ([]linebot.SendingMessage, error)

func (svc *replyImpl) JoinGroupReply(ctx context.Context) ([]linebot.SendingMessage, error) {
	if svc.joinGroupReplyFunc != nil {
		return svc.joinGroupReplyFunc(ctx)
	}
	flex := flexmsg.WelcomeFlex{}
	msg, err := flex.Build()
	if err != nil {
		return nil, err
	}
	return []linebot.SendingMessage{
		msg,
	}, nil
}
