package llm

import (
	"context"
	"slices"
	"time"

	"github.com/94peter/botreplyer/session"

	"github.com/gin-contrib/sessions"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/tmc/langchaingo/llms/googleai"
)

type LLMReply interface {
	MessageTextReply(
		ctx context.Context, typ linebot.EventSourceType,
		groupID, userID, msg string, mysession sessions.Session,
	) ([]linebot.SendingMessage, error)
}

type LLMReplyConfig struct {
	Model           string
	APIKey          string
	ConfigFile      string
	MongoURI        string
	MongoDB         string
	MongoCollection string
	MCPBaseUrls     []string
	AllowMsgType    []linebot.EventSourceType
	AllowUser       bool
}

type LLMReplyOption func(cfg *LLMReplyConfig)

func WithModel(model string) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.Model = model
	}
}

func WithAPIKey(apiKey string) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.APIKey = apiKey
	}
}

func WithConfigFile(configFile string) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.ConfigFile = configFile
	}
}

func WithMCPBaseUrls(mcpBaseUrls []string) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.MCPBaseUrls = mcpBaseUrls
	}
}

func WithAllowMsgType(allowMsgType []linebot.EventSourceType) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.AllowMsgType = allowMsgType
	}
}

func WithAllowUser(allowUser bool) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.AllowUser = allowUser
	}
}

func WithMongo(mongoURI string, db, collection string) LLMReplyOption {
	return func(cfg *LLMReplyConfig) {
		cfg.MongoURI = mongoURI
		cfg.MongoDB = db
		cfg.MongoCollection = collection
	}
}

type llmTextReply struct {
	mgr          ConversationMgr
	allowMsgType []linebot.EventSourceType
	allowUser    bool
}

func NewLLMTextReply(
	ctx context.Context, opts ...LLMReplyOption,
) (LLMReply, error) {
	cfg := &LLMReplyConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	llmCtx, llmCancel := context.WithCancel(ctx)
	defer llmCancel()
	llmmodel, err := googleai.New(llmCtx,
		googleai.WithAPIKey(cfg.APIKey),
		googleai.WithDefaultModel(cfg.Model))

	if err != nil {
		return nil, err
	}
	conversationMgr, err := NewConversationMgr(
		llmmodel,
		cfg.ConfigFile,
		cfg.MCPBaseUrls,
		WithConversationMemoryMongo(cfg.MongoURI, cfg.MongoDB, cfg.MongoCollection),
	)
	if err != nil {
		return nil, err
	}
	return &llmTextReply{
		allowMsgType: cfg.AllowMsgType,
		allowUser:    cfg.AllowUser,
		mgr:          conversationMgr,
	}, nil
}

func (r *llmTextReply) MessageTextReply(
	ctx context.Context, typ linebot.EventSourceType,
	_, userID, msg string, mysession sessions.Session,
) ([]linebot.SendingMessage, error) {
	if !r.allowUser && !session.IsAdmin(mysession) {
		return nil, nil
	}
	if len(r.allowMsgType) > 0 && !slices.Contains(r.allowMsgType, typ) {
		return nil, nil
	}
	conversation, err := r.mgr.NewConversation(ctx, userID)
	if err != nil {
		return nil, err
	}

	today := time.Now()
	response, err := conversation.Chat(ctx, msg, map[string]any{
		"line_user_id": userID,
		"today":        today.Format(time.RFC3339),
		"weekday":      today.Weekday().String(),
	})
	if err != nil {
		return nil, err
	}
	return []linebot.SendingMessage{
		linebot.NewTextMessage(response),
	}, nil
}
