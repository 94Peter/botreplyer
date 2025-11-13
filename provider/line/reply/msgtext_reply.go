package reply

import (
	"context"
	"strings"

	"github.com/arwoosa/vulpes/log"
	"github.com/gin-contrib/sessions"
	"github.com/line/line-bot-sdk-go/v7/linebot"

	"github.com/94peter/botreplyer/provider/line/reply/textreply"
)

type ReplyOption func(r *replyImpl)

func WithTextReply(replies ...textreply.LineKeywordReply) ReplyOption {
	return func(r *replyImpl) {
		r.keywordReplySlice = replies
	}
}

func NewReply(
	replyOptions ...ReplyOption,
) Reply {
	impl := &replyImpl{}
	for _, opt := range replyOptions {
		opt(impl)
	}
	return impl
}

type replyImpl struct {
	keywordReplySlice []textreply.LineKeywordReply
}

func (svc *replyImpl) MessageTextReply(ctx context.Context, typ linebot.EventSourceType, groupID, userID, msg string, session sessions.Session) ([]linebot.SendingMessage, textreply.DelayedMessage, error) {
	defer func() {
		err := session.Save()
		if err != nil {
			log.Errorf("upsert session error: %v", err)
		}
	}()
	msg = strings.Trim(msg, " ")
	for _, reply := range svc.keywordReplySlice {
		msgs, delayedMsg, err := reply.MessageTextReply(ctx, typ, groupID, userID, msg, session)
		if err != nil {
			return nil, nil, err
		}
		if len(msgs) > 0 {
			return msgs, delayedMsg, nil
		}
	}
	return nil, nil, nil
}
