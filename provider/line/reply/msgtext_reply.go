package reply

import (
	"context"
	"reflect"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/94peter/botreplyer/provider/line/reply/textreply"
)

type ReplyOption func(r *replyImpl)

func WithTextReply(replies ...textreply.LineKeywordReply) ReplyOption {
	return func(r *replyImpl) {
		r.keywordReplySlice = replies
	}
}

func WithJoinGroupReply(f JoinGroupReplyFunc) ReplyOption {
	return func(r *replyImpl) {
		r.joinGroupReplyFunc = f
	}
}

func WithTracer(tracer trace.Tracer) ReplyOption {
	return func(r *replyImpl) {
		r.tracer = tracer
	}
}

func NewReply(
	replyOptions ...ReplyOption,
) Reply {
	impl := &replyImpl{}
	for _, opt := range replyOptions {
		opt(impl)
	}
	if impl.tracer == nil {
		impl.tracer = noop.NewTracerProvider().Tracer("lintbot.reply")
	}
	return impl
}

type replyImpl struct {
	keywordReplySlice  []textreply.LineKeywordReply
	joinGroupReplyFunc JoinGroupReplyFunc
	tracer             trace.Tracer
}

func (svc *replyImpl) MessageTextReply(ctx context.Context, typ linebot.EventSourceType, groupID, userID, msg string, session sessions.Session) ([]linebot.SendingMessage, textreply.DelayedMessage, error) {
	msg = strings.Trim(msg, " ")
	ctx, span := svc.startTraceSpan(ctx, "message_reply")
	defer span.End()
	for _, reply := range svc.keywordReplySlice {
		t := reflect.TypeOf(reply)
		ctx, replySpan := svc.startTraceSpan(ctx, t.String())
		defer replySpan.End()
		msgs, delayedMsg, err := reply.MessageTextReply(ctx, typ, groupID, userID, msg, session)
		if err != nil {
			return nil, nil, spanErrorHandler(err, replySpan)
		}
		if len(msgs) > 0 {
			return msgs, delayedMsg, spanErrorHandler(nil, replySpan)
		}
		replySpan.SetStatus(codes.Ok, "ok")
	}
	return nil, nil, nil
}

func (m *replyImpl) startTraceSpan(ctx context.Context, name string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal))
	span.SetAttributes(
		append([]attribute.KeyValue{
			attribute.String("line.bot", string(linebot.EventTypeMessage)),
		}, attributes...)...,
	)
	return ctx, span
}

func spanErrorHandler(err error, span trace.Span) error {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "ok")
	}
	return err
}
