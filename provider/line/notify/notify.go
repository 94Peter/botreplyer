package notify

import (
	"context"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type NotificationContent struct {
	UserIDs string
	Message []linebot.SendingMessage
}

type LineNotify interface {
	GetNotification(ctx context.Context) []*NotificationContent
}

type LineNotificationService interface {
	RegisterNotification(name string, notify LineNotify)
	GetNotification(ctx context.Context, name string) []*NotificationContent
}

type lineNotificationService struct {
	notifyMap map[string]LineNotify
}

func NewLineNotificationService() LineNotificationService {
	return &lineNotificationService{
		notifyMap: make(map[string]LineNotify),
	}
}

func (s *lineNotificationService) RegisterNotification(name string, notify LineNotify) {
	s.notifyMap[name] = notify
}

func (s *lineNotificationService) GetNotification(ctx context.Context, name string) []*NotificationContent {
	if s.notifyMap[name] == nil {
		return nil
	}
	return s.notifyMap[name].GetNotification(ctx)
}
