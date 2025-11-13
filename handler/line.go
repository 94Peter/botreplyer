package handler

import (
	"context"
	"sync"
	"time"

	"github.com/arwoosa/vulpes/ezapi"
	"github.com/arwoosa/vulpes/ezapi/session/store"
	"github.com/arwoosa/vulpes/log"
	"github.com/gin-gonic/gin"
	"github.com/line/line-bot-sdk-go/v7/linebot"

	"github.com/94peter/botreplyer/follow"
	"github.com/94peter/botreplyer/group"
	"github.com/94peter/botreplyer/provider/line"
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

var initLinebotWebhookOnce sync.Once

func InitLinebotWebhook(opts ...linebotWebhookAPIOption) {
	initLinebotWebhookOnce.Do(func() {
		api := &linebotWebhookAPI{}
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
		ezapi.RegisterGinApi(func(r ezapi.Router) {
			// health check
			r.GET("/linebot", api.getEndpoint)
			// linebot webhook
			r.POST("/line", api.webhook)
			r.GET("/follow/is-admin", api.isAdmin)
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
			ctx, cancel := context.WithTimeout(c, time.Second)
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
			ctx, cancel := context.WithTimeout(c, time.Second)
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
			ctx, cancel := context.WithTimeout(c, time.Second)
			defer cancel()
			err := d.followStore.Delete(ctx, event.Source.UserID)
			if err != nil {
				log.Warnf("Failed to add follow: %v", err)
				continue
			}
		case linebot.EventTypeFollow:
			log.Infof("follow: %s", event.Source.UserID)
			ctx, cancel := context.WithTimeout(c, time.Second)
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
			msgReplies, err := d.svc.WelcomeReply(c, event.Source.UserID)
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
					follow, err := d.followStore.Get(c, userId)
					if err != nil {
						ginSession.Set(session.KeyIsAdmin, false)
					} else {
						ginSession.Set(session.KeyIsAdmin, follow.IsAdmin())
					}
				}

				// Echo the same message back to the user
				msgReplies, delayedMsg, err := d.svc.MessageTextReply(c, typ, groupId, userId, message.Text, ginSession)
				if err != nil {
					log.Warnf("Failed to get reply: %v", err)
				}
				if len(msgReplies) == 0 {
					continue
				}
				if _, err = d.bot.ReplyMessage(event.ReplyToken, msgReplies...); err != nil {
					log.Warnf("Failed to reply: %v", err)
				}
				if delayedMsg != nil {
					msgs := <-delayedMsg
					if _, err = d.bot.PushMessage(event.Source.UserID, msgs...); err != nil {
						log.Warnf("Failed to push: %v", err)
					}
				}

			}
		}
	}
	c.Writer.WriteHeader(200)
}
