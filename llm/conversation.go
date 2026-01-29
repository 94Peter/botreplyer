package llm

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/memory/mongo"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/tools"
)

type ConversationPromptConfig struct {
	SystemPrompt *struct {
		Template string   `yaml:"template"`
		Inputs   []string `yaml:"input_variables"`
	} `yaml:"system_prompt"`
}

type MemoryMongoCfg struct {
	ConnectionURL  string `yaml:"connection_url"`
	DatabaseName   string `yaml:"database_name"`
	CollectionName string `yaml:"collection_name"`
}

type ConversationMgr interface {
	NewConversation(ctx context.Context, sessionID string) (Conversation, error)
}

type conversationMgr struct {
	agent        *agents.ConversationalAgent
	chatTemplate prompts.ChatPromptTemplate

	mongoMemory *MemoryMongoCfg

	conversationPool map[string]Conversation
	mu               sync.RWMutex
}

func NewConversationMgr(
	llmmodel llms.Model, configFile string, mcpBaseUrls []string,
	opts ...ConversationMgrOption,
) (ConversationMgr, error) {
	var cfx ConversationPromptConfig
	bytes, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(bytes, &cfx)
	if err != nil {
		return nil, err
	}
	chatTemplate := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			cfx.SystemPrompt.Template,
			cfx.SystemPrompt.Inputs,
		),
		prompts.NewHumanMessagePromptTemplate(
			"{{.user_input}}",
			[]string{"user_input"},
		),
	})

	var mcpTools []tools.Tool
	for _, url := range mcpBaseUrls {
		tools, err := newMcpTools(url)
		if err != nil {
			return nil, err
		}
		mcpTools = append(mcpTools, tools...)
	}
	agent := agents.NewConversationalAgent(llmmodel, mcpTools)
	mgr := &conversationMgr{agent: agent, chatTemplate: chatTemplate, conversationPool: make(map[string]Conversation)}
	for _, opt := range opts {
		opt(mgr)
	}
	return mgr, nil
}

func (c *conversationMgr) NewConversation(ctx context.Context, sessionID string) (Conversation, error) {
	if c.conversationPool[sessionID] != nil {
		return c.conversationPool[sessionID], nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	chatHistory, err := mongo.NewMongoDBChatMessageHistory(ctx,
		mongo.WithConnectionURL(c.mongoMemory.ConnectionURL),
		mongo.WithDataBaseName(c.mongoMemory.DatabaseName),
		mongo.WithCollectionName(c.mongoMemory.CollectionName),
		mongo.WithSessionID(sessionID),
	)
	if err != nil {
		return nil, err
	}

	executor := agents.NewExecutor(c.agent, agents.WithMemory(memory.NewConversationBuffer(
		memory.WithChatHistory(chatHistory),
	)))
	con := &conversation{chatTemplate: c.chatTemplate, chain: executor}
	c.conversationPool[sessionID] = con
	return con, nil
}

func newMcpTools(baseUrl string) ([]tools.Tool, error) {
	mcpClt, err := client.NewStreamableHttpClient(baseUrl)
	if err != nil {
		return nil, err
	}

	adapter, err := langchaingo_mcp_adapter.New(mcpClt)
	if err != nil {
		return nil, err
	}

	// Get all tools from MCP server
	return adapter.Tools()
}

type Conversation interface {
	Chat(ctx context.Context, message string, inputs map[string]any) (string, error)
}

type conversation struct {
	chain        chains.Chain
	chatTemplate prompts.ChatPromptTemplate
}

func (c *conversation) Chat(ctx context.Context, message string, inputs map[string]any) (string, error) {
	inputs["user_input"] = message
	prompt, err := c.chatTemplate.Format(inputs)
	if err != nil {
		return "", err
	}
	return chains.Run(ctx, c.chain, prompt)
}
