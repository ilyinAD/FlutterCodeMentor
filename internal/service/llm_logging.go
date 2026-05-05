package service

import (
	"context"

	"go.uber.org/zap"
)

type LoggingLLMClient struct {
	inner  LLMClient
	logger *zap.Logger
}

func NewLoggingLLMClient(inner LLMClient, logger *zap.Logger) *LoggingLLMClient {
	return &LoggingLLMClient{
		inner:  inner,
		logger: logger,
	}
}

func (c *LoggingLLMClient) ModelName() string {
	return c.inner.ModelName()
}

func (c *LoggingLLMClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	c.logger.Info("LLM request",
		zap.String("model", c.inner.ModelName()),
		zap.String("system_prompt", systemPrompt),
		zap.String("user_prompt", userPrompt),
	)

	response, err := c.inner.Chat(ctx, systemPrompt, userPrompt)

	if err != nil {
		c.logger.Warn("LLM error",
			zap.String("model", c.inner.ModelName()),
			zap.Error(err),
		)
		return response, err
	}

	c.logger.Info("LLM response",
		zap.String("model", c.inner.ModelName()),
		zap.String("response", response),
	)

	return response, nil
}
