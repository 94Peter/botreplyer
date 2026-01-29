package handler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/94peter/vulpes/ezapi"
	"github.com/94peter/vulpes/ezapi/session/store"
	"github.com/94peter/vulpes/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/94peter/botreplyer/follow"
	"github.com/94peter/botreplyer/group"
	"github.com/94peter/botreplyer/provider/line"
	"github.com/94peter/botreplyer/provider/line/notify"
	"github.com/94peter/botreplyer/provider/line/reply"
	"github.com/94peter/botreplyer/session"
)

type linebotWebhookAPIOption func(api *linebotWebhookAPI)

func WithLineSDK(sdk line.SDK) linebotWebhookAPIOption {
	return func(api *linebotWebhookAPI) {
		api.bot = sdk
	}
}

func WithLineMsgReplyService(svc reply.Reply) linebotWebhookAPIOption {
	return func(api *linebotWebhookAPI) {
		api.svc = svc
	}
}

func WithFollowStore(store follow.Store) linebotWebhookAPIOption {
	return func(api *linebotWebhookAPI) {
		api.followStore = store
	}
}

func WithAdminUserId(userId string) linebotWebhookAPIOption {
	return func(api *linebotWebhookAPI) {
		api.adminUserId = userId
	}
}

func WithGroupStore(store group.Store) linebotWebhookAPIOption {
	return func(api *linebotWebhookAPI) {
		api.groupStore = store
	}
}

func WithLineNotificationService(notifySvc notify.LineNotificationService) linebotWebhookAPIOption {
	return func(api *linebotWebhookAPI) {
		api.notifySvc = notifySvc
	}
}

var initLinebotWebhookOnce sync.Once

func InitLinebotWebhook(opts ...linebotWebhookAPIOption) {
	initLinebotWebhookOnce.Do(func() {
		replyTracer := otel.Tracer("botReply_handler")
		api := &linebotWebhookAPI{tracer: replyTracer}
		for _, opt := range opts {
			opt(api)
		}
		if api.bot == nil {
			panic("bot is nil")
		}
		if api.svc == nil {
			panic("svc is nil")
		}
		if api.followStore == nil {
			panic("follow store is nil")
		}
		if api.adminUserId == "" {
			panic("admin user id is empty")
		}
		if api.groupStore == nil {
			panic("group store is nil")
		}
		ezapi.RegisterSessionInjector(api)
		ezapi.RegisterGinApi(func(r ezapi.RouterGroup) {
			// linebot webhook
			r.POST("/line", api.webhook)
			r.GET("/follow/is-admin", api.isAdmin)
			r.POST("/line/notification/:type", api.notification)
		})
	})
}

type linebotWebhookAPI struct {
	sessionStore store.Store
	svc          reply.Reply
	bot          line.SDK
	followStore  follow.Store
	groupStore   group.Store
	tracer       trace.Tracer
	notifySvc    notify.LineNotificationService
	cookieName   string
	adminUserId  string
}

func (d *linebotWebhookAPI) InjectSessionStore(store store.Store, cookieName string) {
	d.sessionStore = store
	d.cookieName = cookieName
}

func (d *linebotWebhookAPI) isAdmin(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		c.JSON(http.StatusOK, gin.H{"isAdmin": false})
		return
	}
	follow, err := d.followStore.Get(c, userId)
	if err != nil {
		log.Err(err)
		c.JSON(http.StatusOK, gin.H{"isAdmin": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"isAdmin": follow.IsAdmin()})
}

func (d *linebotWebhookAPI) notification(c *gin.Context) {
	if d.notifySvc == nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "notify service not set"})
		return
	}
	typ := c.Param("type")
	contents := d.notifySvc.GetNotification(c.Request.Context(), typ)
	_, pushspan := d.tracer.Start(c.Request.Context(), "bot_push_message", trace.WithSpanKind(trace.SpanKindClient))
	maxWorkers := 5 // 控制併發數量
	sem := make(chan struct{}, maxWorkers)
	wg := sync.WaitGroup{}
	pushspan.SetAttributes(attribute.Int("message_number", len(contents)))
	for _, content := range contents {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if _, err := d.bot.PushMessage(content.UserIDs, content.Message...); err != nil {
				log.Warnf("Failed to push: %v", err)
			}
		}()
	}
	wg.Wait()
	pushspan.End()
	c.Writer.WriteHeader(http.StatusOK)
}

func (d *linebotWebhookAPI) webhook(c *gin.Context) {
	events, err := d.bot.ParseRequest(c.Request)

	if err != nil {
		if errors.Is(err, linebot.ErrInvalidSignature) {
			c.Writer.WriteHeader(http.StatusBadRequest)
		} else {
			c.Writer.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	for _, event := range events {
		switch event.Type {
		case linebot.EventTypeLeave:
			d.handleLeaveEvent(c, event)
		case linebot.EventTypeJoin:
			d.handleJoinEvent(c, event)
		case linebot.EventTypeUnfollow:
			d.handleUnfollowEvent(c, event)
		case linebot.EventTypeFollow:
			d.handleFollowEvent(c, event)
		case linebot.EventTypeMessage:
			d.handleMessageEvent(c, event)
		}
	}
	c.Writer.WriteHeader(http.StatusOK)
}

func (d *linebotWebhookAPI) handleLeaveEvent(c *gin.Context, event *linebot.Event) {
	// 離開群組的行為
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer cancel()
	err := d.groupStore.Delete(ctx, event.Source.GroupID)
	if err != nil {
		log.Warnf("Failed to add follow: %v", err)
		return
	}
}

func (d *linebotWebhookAPI) handleJoinEvent(c *gin.Context, event *linebot.Event) {
	// 加入群組的行為
	active := true
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer cancel()
	err := d.groupStore.Add(ctx, event.Source.GroupID, active)
	if err != nil {
		log.Warnf("Failed to add follow: %v", err)
		return
	}
	msgReplies, err := d.svc.JoinGroupReply(c)
	if err != nil {
		log.Warnf("Failed to get welcome reply: %v", err)
	}
	if len(msgReplies) == 0 {
		return
	}
	if _, err = d.bot.ReplyMessage(event.ReplyToken, msgReplies...); err != nil {
		log.Warnf("Failed to reply: %v", err)
	}
}

func (d *linebotWebhookAPI) handleUnfollowEvent(c *gin.Context, event *linebot.Event) {
	// 取消追蹤的行為
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer cancel()
	err := d.followStore.Delete(ctx, event.Source.UserID)
	if err != nil {
		log.Warnf("Failed to add follow: %v", err)
		return
	}
}

func (d *linebotWebhookAPI) handleFollowEvent(c *gin.Context, event *linebot.Event) {
	log.Infof("follow: %s", event.Source.UserID)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer cancel()
	userInfo, err := d.bot.GetUserInfo(event.Source.UserID)
	if err != nil {
		log.Warnf("Failed to get user info: %v", err)
		return
	}
	err = d.followStore.Add(ctx, event.Source.UserID, userInfo.Name, d.adminUserId == event.Source.UserID)
	if err != nil {
		log.Warnf("Failed to add follow: %v", err)
		return
	}
	msgReplies, err := d.svc.WelcomeReply(c.Request.Context(), event.Source.UserID)
	if err != nil {
		log.Warnf("Failed to get welcome reply: %v", err)
	}
	if len(msgReplies) == 0 {
		return
	}
	if _, err = d.bot.ReplyMessage(event.ReplyToken, msgReplies...); err != nil {
		log.Warnf("Failed to reply: %v", err)
	}
}

func (d *linebotWebhookAPI) handleMessageEvent(c *gin.Context, event *linebot.Event) {
	switch message := event.Message.(type) {
	case *linebot.ImageMessage:
		log.Debug("line msg: image")
	case *linebot.VideoMessage:
		log.Debug("line msg: video")
	case *linebot.AudioMessage:
		log.Debug("line msg: audio")
	case *linebot.LocationMessage:
		log.Debug("line msg: location")
	case *linebot.TextMessage:
		d.handleTextMsgEvent(c, event, message)
	}
}

func (d *linebotWebhookAPI) handleTextMsgEvent(c *gin.Context, event *linebot.Event, message *linebot.TextMessage) {
	log.Debug("line msg: " + message.Text)
	name := d.cookieName
	userId := event.Source.UserID
	typ := event.Source.Type
	groupId := event.Source.GroupID
	token, err := d.sessionStore.EncodeToken(name, session.UserIdToObjectID(userId))
	if err != nil {
		log.Warnf("Failed to encode token: %v", err)
		return
	}
	c.Request.Header.Add("X-"+name, token)

	ginSession, saveSession, err := d.getSession(c, name, userId)
	if err != nil {
		log.Warnf("Failed to get session: %v", err)
		return
	}
	defer saveSession()

	if !session.IsKeyExist(ginSession, session.KeyIsAdmin) {
		follow, err := d.followStore.Get(c.Request.Context(), userId)
		if err != nil {
			ginSession.Set(session.KeyIsAdmin, false)
		} else {
			ginSession.Set(session.KeyIsAdmin, follow.IsAdmin())
		}
	}

	// Echo the same message back to the user
	msgReplies, delayedMsg, err := d.svc.MessageTextReply(
		c.Request.Context(), typ, groupId, userId, message.Text, ginSession)
	if err != nil {
		log.Warnf("Failed to get reply: %v", err)
	}
	if len(msgReplies) == 0 {
		return
	}
	_, span := d.tracer.Start(c.Request.Context(), "bot_reply_message", trace.WithSpanKind(trace.SpanKindClient))
	if _, err = d.bot.ReplyMessage(event.ReplyToken, msgReplies...); err != nil {
		log.Warnf("Failed to reply: %v", err)
	}
	span.End()

	if delayedMsg != nil {
		msgs := <-delayedMsg
		_, pushspan := d.tracer.Start(c.Request.Context(), "bot_push_message", trace.WithSpanKind(trace.SpanKindClient))
		if _, err = d.bot.PushMessage(event.Source.UserID, msgs...); err != nil {
			log.Warnf("Failed to push: %v", err)
		}
		pushspan.End()
	}
}

func (d *linebotWebhookAPI) getSession(c *gin.Context, name, userId string) (sessions.Session, func(), error) {
	newSession, err := d.sessionStore.New(c.Request, name)
	if err != nil {
		log.Warnf("Failed to new session: %v", err)
		return nil, nil, err
	}
	if newSession.ID == "" {
		newSession.ID = session.UserIdToObjectID(userId)
	}
	ginSession := session.NewSessionFromSession(c, name, newSession, d.sessionStore)
	return ginSession, func() {
		err := ginSession.Save()
		if err != nil {
			log.Warnf("Failed to save session: %v", err)
		}
	}, nil
}
