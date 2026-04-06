package llm

import (
	"context"
	"fmt"
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
	"github.com/tmc/langchaingo/schema"
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
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	err = yaml.Unmarshal(bytes, &cfx)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Prepare chat template using system prompt from config
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
			return nil, fmt.Errorf("failed to create MCP tools for %s: %w", url, err)
		}
		mcpTools = append(mcpTools, tools...)
	}

	// Create the agent.
	// Note: NewConversationalAgent uses a default ReAct prompt.
	// We are relying on injecting our system prompt via the user input formatting in Chat()
	// to provide specific instructions to the model.
	agent := agents.NewConversationalAgent(llmmodel, mcpTools)

	mgr := &conversationMgr{
		agent:            agent,
		chatTemplate:     chatTemplate,
		conversationPool: make(map[string]Conversation),
	}
	for _, opt := range opts {
		opt(mgr)
	}
	return mgr, nil
}

func (c *conversationMgr) NewConversation(ctx context.Context, sessionID string) (Conversation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if con, ok := c.conversationPool[sessionID]; ok {
		return con, nil
	}

	var chatHistory schema.ChatMessageHistory
	if c.mongoMemory != nil {
		var err error
		chatHistory, err = mongo.NewMongoDBChatMessageHistory(ctx,
			mongo.WithConnectionURL(c.mongoMemory.ConnectionURL),
			mongo.WithDataBaseName(c.mongoMemory.DatabaseName),
			mongo.WithCollectionName(c.mongoMemory.CollectionName),
			mongo.WithSessionID(sessionID),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create mongo history: %w", err)
		}
	} else {
		// Fallback to in-memory history if mongo is not configured
		chatHistory = memory.NewChatMessageHistory()
	}

	executor := agents.NewExecutor(
		c.agent,
		agents.WithMemory(memory.NewConversationBuffer(
			memory.WithChatHistory(chatHistory),
		)),
	)

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
	if inputs == nil {
		inputs = make(map[string]any)
	}
	inputs["user_input"] = message

	// Format the full prompt (System + User) using the template
	prompt, err := c.chatTemplate.Format(inputs)
	if err != nil {
		return "", fmt.Errorf("failed to format chat template: %w", err)
	}

	result, err := chains.Run(ctx, c.chain, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to run chain: %w", err)
	}
	return result, nil
}
