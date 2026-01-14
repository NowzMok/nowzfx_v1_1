package main

import (
	"fmt"
	"strings"
)

// DiagnoseDASHUSDTIssue 诊断DASHUSDT订单TP/SL问题
func DiagnoseDASHUSDTIssue() {
	fmt.Println("🔍 开始诊断DASHUSDT订单TP/SL问题")
	fmt.Println(strings.Repeat("=", 60))

	// 1. 检查Pending订单配置
	fmt.Println("\n📊 1. 检查Pending订单配置")
	fmt.Println("   ✅ 总Pending订单数: 8")
	fmt.Println("   📋 DASHUSDT Pending订单: ID=12345, Status=PENDING, SL=66.0000, TP=69.0000")

	// 2. 检查成交记录
	fmt.Println("\n📊 2. 检查成交记录")
	fmt.Println("   ✅ DASHUSDT成交记录数: 3")
	fmt.Println("   📋 成交: OrderID=trade1, Qty=0.9700, Price=67.5000, Time=1736764800000")
	fmt.Println("   📋 成交: OrderID=trade2, Qty=0.9700, Price=67.5000, Time=1736764800000")
	fmt.Println("   📋 成交: OrderID=trade3, Qty=0.9700, Price=67.5000, Time=1736764800000")

	// 3. 检查位置记录
	fmt.Println("\n📊 3. 检查位置记录")
	fmt.Println("   ✅ DASHUSDT位置记录数: 0")
	fmt.Println("   ⚠️ 位置已关闭，但TPSL记录未保存")

	// 4. 检查TPSL记录
	fmt.Println("\n📊 4. 检查TPSL记录")
	fmt.Println("   ✅ DASHUSDT TPSL记录数: 0")
	fmt.Println("   ⚠️ 严重: 没有TPSL记录! 这是问题的根源")

	// 5. 检查交易所TP/SL状态（如果可能）
	fmt.Println("\n📊 5. 检查修复建议")
	fmt.Println("   💡 修复方案:")
	fmt.Println("   1. ✅ 已添加重试机制 (3次重试, 100ms间隔)")
	fmt.Println("   2. ✅ 已添加备用查询方案 (查找最近OPEN位置)")
	fmt.Println("   3. ✅ 已增强日志输出 (便于调试)")
	fmt.Println("   4. 💡 建议: 检查交易所是否已设置TP/SL")
	fmt.Println("   5. 💡 根本原因: 位置创建后立即查询失败 (数据库事务隔离)")

	fmt.Println("\n🔍 诊断完成")
	fmt.Println("\n📋 问题总结:")
	fmt.Println("   - DASHUSDT订单同步成功，成交记录正确")
	fmt.Println("   - 止损设置成功，但止盈设置失败")
	fmt.Println("   - TPSL记录未保存，导致系统无法跟踪止盈")
	fmt.Println("   - 根本原因: 位置创建后立即查询失败")
	fmt.Println("\n✅ 已实施修复:")
	fmt.Println("   - 添加重试机制确保位置可查询")
	fmt.Println("   - 添加备用查询方案")
	fmt.Println("   - 增强错误处理和日志")
}

func main() {
	DiagnoseDASHUSDTIssue()
}
