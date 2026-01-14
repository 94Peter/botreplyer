package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/94peter/vulpes/ezapi"
	"github.com/94peter/vulpes/ezapi/session/store"
	"github.com/94peter/vulpes/log"
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
			// health check
			r.GET("/linebot", api.getEndpoint)
			// linebot webhook
			r.POST("/line", api.webhook)
			r.GET("/follow/is-admin", api.isAdmin)
			r.POST("/line/notification/:type", api.notification)
		})
	})
}

type linebotWebhookAPI struct {
	sessionStore store.Store
	cookieName   string
	svc          reply.Reply
	bot          line.SDK
	followStore  follow.Store
	groupStore   group.Store
	adminUserId  string
	tracer       trace.Tracer
	notifySvc    notify.LineNotificationService
}

func (d *linebotWebhookAPI) InjectSessionStore(store store.Store, cookieName string) {
	d.sessionStore = store
	d.cookieName = cookieName
}

func (d *linebotWebhookAPI) isAdmin(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		c.JSON(200, gin.H{"isAdmin": false})
		return
	}
	follow, err := d.followStore.Get(c, userId)
	if err != nil {
		log.Err(err)
		c.JSON(200, gin.H{"isAdmin": false})
		return
	}
	c.JSON(200, gin.H{"isAdmin": follow.IsAdmin()})
}

func (d *linebotWebhookAPI) notification(c *gin.Context) {
	if d.notifySvc == nil {
		c.Writer.WriteHeader(500)
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
	c.Writer.WriteHeader(200)
}

func (d *linebotWebhookAPI) getEndpoint(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Hello World"})
}

func (d *linebotWebhookAPI) webhook(c *gin.Context) {
	events, err := d.bot.ParseRequest(c.Request)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			c.Writer.WriteHeader(400)
		} else {
			c.Writer.WriteHeader(500)
		}
		return
	}
	for _, event := range events {
		switch event.Type {
		case linebot.EventTypeLeave:
			// 離開群組的行為
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			err := d.groupStore.Delete(ctx, event.Source.GroupID)
			if err != nil {
				log.Warnf("Failed to add follow: %v", err)
				continue
			}
		case linebot.EventTypeJoin:
			// 加入群組的行為
			// TODO: 加上判斷允許群組數量是否達到上限
			active := true
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			err := d.groupStore.Add(ctx, event.Source.GroupID, active)
			if err != nil {
				log.Warnf("Failed to add follow: %v", err)
				continue
			}
			msgReplies, err := d.svc.JoinGroupReply(c)
			if err != nil {
				log.Warnf("Failed to get welcome reply: %v", err)
			}
			if len(msgReplies) == 0 {
				continue
			}
			if _, err = d.bot.ReplyMessage(event.ReplyToken, msgReplies...); err != nil {
				log.Warnf("Failed to reply: %v", err)
			}
		case linebot.EventTypeUnfollow:
			// 取消追蹤的行為
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			err := d.followStore.Delete(ctx, event.Source.UserID)
			if err != nil {
				log.Warnf("Failed to add follow: %v", err)
				continue
			}
		case linebot.EventTypeFollow:
			log.Infof("follow: %s", event.Source.UserID)
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			userInfo, err := d.bot.GetUserInfo(event.Source.UserID)
			if err != nil {
				log.Warnf("Failed to get user info: %v", err)
				continue
			}
			err = d.followStore.Add(ctx, event.Source.UserID, userInfo.Name, d.adminUserId == event.Source.UserID)
			if err != nil {
				log.Warnf("Failed to add follow: %v", err)
				continue
			}
			msgReplies, err := d.svc.WelcomeReply(c.Request.Context(), event.Source.UserID)
			if err != nil {
				log.Warnf("Failed to get welcome reply: %v", err)
			}
			if len(msgReplies) == 0 {
				continue
			}
			if _, err = d.bot.ReplyMessage(event.ReplyToken, msgReplies...); err != nil {
				log.Warnf("Failed to reply: %v", err)
			}
		case linebot.EventTypeMessage:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				log.Debug("line msg: " + message.Text)
				name := d.cookieName
				userId := event.Source.UserID
				typ := event.Source.Type
				groupId := event.Source.GroupID
				token, err := d.sessionStore.EncodeToken(name, session.UserIdToObjectID(userId))
				if err != nil {
					log.Warnf("Failed to encode token: %v", err)
					continue
				}
				c.Request.Header.Add("X-"+d.cookieName, token)
				newSession, err := d.sessionStore.New(c.Request, name)
				if err != nil {
					log.Warnf("Failed to new session: %v", err)
					continue
				}
				if newSession.ID == "" {
					newSession.ID = session.UserIdToObjectID(userId)
				}
				ginSession := session.NewSessionFromSession(c, name, newSession, d.sessionStore)
				defer func() {
					err := ginSession.Save()
					if err != nil {
						log.Warnf("Failed to save session: %v", err)
					}
				}()

				if !session.IsKeyExist(ginSession, session.KeyIsAdmin) {
					follow, err := d.followStore.Get(c.Request.Context(), userId)
					if err != nil {
						ginSession.Set(session.KeyIsAdmin, false)
					} else {
						ginSession.Set(session.KeyIsAdmin, follow.IsAdmin())
					}
				}

				// Echo the same message back to the user
				msgReplies, delayedMsg, err := d.svc.MessageTextReply(c.Request.Context(), typ, groupId, userId, message.Text, ginSession)
				if err != nil {
					log.Warnf("Failed to get reply: %v", err)
				}
				if len(msgReplies) == 0 {
					continue
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
		}
	}
	c.Writer.WriteHeader(200)
}
