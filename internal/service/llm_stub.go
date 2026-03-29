package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const stubResponse = `{
  "overall_status": "needs_improvement",
  "confidence": 0.85,
  "feedbacks": [
    {
      "type": "style_issue",
      "file_path": "lib/main.dart",
      "line_start": 1,
      "line_end": 5,
      "code_snippet": "void main() => runApp(MyApp());",
      "suggested_fix": "void main() {\n  runApp(const MyApp());\n}",
      "description": "[STUB] Consider using const constructor for MyApp to optimize widget rebuilds.",
      "severity": 2
    },
    {
      "type": "improvement",
      "file_path": "lib/main.dart",
      "line_start": 10,
      "line_end": 15,
      "code_snippet": "class MyApp extends StatelessWidget {}",
      "suggested_fix": "class MyApp extends StatelessWidget { const MyApp({super.key}); }",
      "description": "[STUB] Add const constructor with key parameter for better widget identity.",
      "severity": 1
    },
    {
      "type": "criterion_check",
      "file_path": "lib/main.dart",
      "line_start": 1,
      "line_end": 1,
      "code_snippet": "",
      "suggested_fix": "",
      "description": "PASS: [STUB] Code compiles and runs without errors.",
      "severity": 1
    },
    {
      "type": "criterion_check",
      "file_path": "lib/main.dart",
      "line_start": 1,
      "line_end": 1,
      "code_snippet": "",
      "suggested_fix": "",
      "description": "FAIL: [STUB] Widget tests are missing for the main screen.",
      "severity": 5
    }
  ]
}`

type StubLLMClient struct {
	delay  time.Duration
	logger *zap.Logger
}

func NewStubLLMClient(logger *zap.Logger) *StubLLMClient {
	return &StubLLMClient{
		delay:  500 * time.Millisecond,
		logger: logger,
	}
}

func (c *StubLLMClient) ModelName() string {
	return "stub"
}

func (c *StubLLMClient) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	c.logger.Info("Stub LLM client: simulating API call",
		zap.Duration("delay", c.delay),
	)

	select {
	case <-time.After(c.delay):
		return stubResponse, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
