package llm

import (
	"fmt"
	"os"
	"testing"
)

// TestMain 在包级别检查必要的环境变量，如果缺失则跳过所有 llm 相关测试。
func TestMain(m *testing.M) {
	appID := os.Getenv("QWEN_APP_ID")
	apiKey := os.Getenv("QWEN_API_KEY")

	if appID == "" || apiKey == "" {
		fmt.Println("[skip] llm tests skipped: QWEN_APP_ID or QWEN_API_KEY not set")
		os.Exit(0)
	}

	os.Exit(m.Run())
}
