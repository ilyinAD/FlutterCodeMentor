package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type LoggingLLMClient struct {
	inner   LLMClient
	logFile string
	mu      sync.Mutex
	logger  *zap.Logger
}

func NewLoggingLLMClient(inner LLMClient, logFile string, logger *zap.Logger) *LoggingLLMClient {
	return &LoggingLLMClient{
		inner:   inner,
		logFile: logFile,
		logger:  logger,
	}
}

func (c *LoggingLLMClient) ModelName() string {
	return c.inner.ModelName()
}

func (c *LoggingLLMClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	c.writeLog(systemPrompt, userPrompt)

	response, err := c.inner.Chat(ctx, systemPrompt, userPrompt)

	if err == nil {
		c.writeResponseLog(response)
	}

	return response, err
}

func (c *LoggingLLMClient) writeLog(systemPrompt, userPrompt string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.OpenFile(c.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.logger.Warn("Failed to open prompt log file", zap.Error(err))
		return
	}
	defer f.Close()

	entry := fmt.Sprintf(
		"========== %s ==========\n"+
			"Model: %s\n\n"+
			"--- SYSTEM PROMPT ---\n%s\n\n"+
			"--- USER PROMPT ---\n%s\n\n",
		time.Now().Format("2006-01-02 15:04:05"),
		c.inner.ModelName(),
		systemPrompt,
		userPrompt,
	)

	f.WriteString(entry)
}

func (c *LoggingLLMClient) writeResponseLog(response string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.OpenFile(c.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	entry := fmt.Sprintf("--- RESPONSE ---\n%s\n\n", response)
	f.WriteString(entry)
}
