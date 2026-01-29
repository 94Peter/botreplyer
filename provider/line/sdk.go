package line

import (
	"errors"
	"net/http"
	"time"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	gocache "github.com/patrickmn/go-cache"
)

type SDK interface {
	GetUserInfo(string) (*UserInfo, error)
	ParseRequest(*http.Request) ([]*linebot.Event, error)
	PushMessage(string, ...linebot.SendingMessage) (*linebot.BasicResponse, error)
	ReplyMessage(replyToken string, messages ...linebot.SendingMessage) (*linebot.BasicResponse, error)
}

type UserInfo struct {
	UserId     string `json:"userId"`
	Name       string `json:"name"`
	PictureUrl string `json:"pictureUrl"`
	Language   string `json:"language"`
}

const (
	defaultCacheExpiration      = 5 * time.Minute
	defaultCacheCleanupInterval = 10 * time.Minute
)

func NewSDK(channelSecret, accessToken string) (SDK, error) {
	bot, err := linebot.New(channelSecret, accessToken)
	if err != nil {
		return nil, err
	}
	return &sdkImpl{
		cache: gocache.New(defaultCacheExpiration, defaultCacheCleanupInterval),
		bot:   bot,
	}, nil
}

type sdkImpl struct {
	cache *gocache.Cache
	bot   *linebot.Client
}

func (s *sdkImpl) ParseRequest(r *http.Request) ([]*linebot.Event, error) {
	return s.bot.ParseRequest(r)
}

func (s *sdkImpl) PushMessage(userId string, messages ...linebot.SendingMessage) (*linebot.BasicResponse, error) {
	return s.bot.PushMessage(userId, messages...).Do()
}

func (s *sdkImpl) ReplyMessage(replyToken string, messages ...linebot.SendingMessage) (*linebot.BasicResponse, error) {
	return s.bot.ReplyMessage(replyToken, messages...).Do()
}

const userInfoCacheKeyPrefix = "user:"

func (s *sdkImpl) GetUserInfo(userId string) (*UserInfo, error) {
	key := userInfoCacheKeyPrefix + userId
	if userInfo, found := s.cache.Get(key); found {
		if userInfo == nil {
			return nil, errors.New("user info not found")
		}
		if userInfo, ok := userInfo.(*UserInfo); ok {
			return userInfo, nil
		}
		return nil, errors.New("invalid user info cache")
	}
	userInfo, err := s.getUserInfoFromSdk(userId)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, userInfo, time.Hour)
	return userInfo, nil
}

func (s *sdkImpl) getUserInfoFromSdk(userId string) (*UserInfo, error) {
	profile, err := s.bot.GetProfile(userId).Do()
	if err != nil {
		return nil, err
	}
	return &UserInfo{
		UserId:     profile.UserID,
		Name:       profile.DisplayName,
		PictureUrl: profile.PictureURL,
		Language:   profile.Language,
	}, nil
}
