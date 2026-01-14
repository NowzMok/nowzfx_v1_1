package alpaca

import (
	"fmt"
	"os"
	"testing"
)

// TestMain 在包级别检查 Alpaca API 凭证环境变量，无则跳过所有测试
func TestMain(m *testing.M) {
	apiKey := os.Getenv("ALPACA_API_KEY")
	if apiKey == "" {
		fmt.Println("[skip] alpaca tests skipped: ALPACA_API_KEY environment variable not set")
		os.Exit(0)
	}

	os.Exit(m.Run())
}
