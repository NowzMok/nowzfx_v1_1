package coinank

import (
	"fmt"
	"os"
	"testing"
)

// TestMain 在包级别检查是否有有效的 API Key，无则跳过所有测试以避免 panic
func TestMain(m *testing.M) {
	// 检查是否有有效的 API Key
	// TestApikey 在 main_test.go 中定义，为空时表示无有效凭证
	if TestApikey == "" {
		fmt.Println("[skip] coinank tests skipped: no valid API key")
		os.Exit(0)
	}

	os.Exit(m.Run())
}
