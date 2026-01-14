# 触发价格状态高亮修复

## 问题描述

用户反馈：在策略工作室选择"剥头皮"或"短线"风格并保存后，重新打开策略时，**选中的风格按钮没有高亮显示**。

## 根本原因

TriggerPriceEditor组件使用`useState`初始化`preset`状态，但**缺少响应配置变化的`useEffect`**。

```typescript
// ❌ 问题代码
const [preset, setPreset] = useState<string>(() => {
  // 初始化时推断预设
  if (config?.style) return config.style
  // ... 其他推断逻辑
})
// 缺少：当config变化时更新preset
```

当用户保存策略后重新打开：
1. `config`参数从数据库加载新的配置
2. 但`preset`状态保持不变（初始化时的值）
3. 导致UI显示与实际配置不一致

## 解决方案

添加`useEffect`来同步`preset`状态与`config`参数：

```typescript
// ✅ 修复后的代码
const [preset, setPreset] = useState<string>(() => {
  // 初始化逻辑保持不变
  if (config?.style) return config.style
  // ...
})

// 新增：响应配置变化
useEffect(() => {
  if (config?.style) {
    setPreset(config.style)
  } else if (config?.pullback_ratio !== undefined && config?.extra_buffer !== undefined) {
    // 根据参数推断预设
    if (config.pullback_ratio === 0.005 && config.extra_buffer === 0.001) {
      setPreset('scalp')
    } else if (config.pullback_ratio === 0.01 && config.extra_buffer === 0.002) {
      setPreset('short_term')
    } else if (config.pullback_ratio === 0.02 && config.extra_buffer === 0.005) {
      setPreset('swing')
    } else if (config.pullback_ratio === 0.05 && config.extra_buffer === 0.01) {
      setPreset('long_term')
    } else {
      setPreset('swing')
    }
  }
}, [config])  // 依赖config，当config变化时触发
```

## 修复效果

### 修复前
```
用户操作：选择"剥头皮" → 保存 → 重新打开
UI显示：所有按钮都不高亮 ❌
实际配置：style="scalp" ✅
```

### 修复后
```
用户操作：选择"剥头皮" → 保存 → 重新打开
UI显示："剥头皮"按钮高亮 ✅
实际配置：style="scalp" ✅
```

## 技术要点

### 1. React状态同步原则
- **初始化**：`useState`的惰性初始化只在组件首次渲染时执行
- **更新**：使用`useEffect`响应外部props变化
- **依赖数组**：明确列出依赖项`[config]`

### 2. 双向同步机制
```typescript
// 用户点击 → 更新config
handlePresetChange = (presetName) => {
  setPreset(presetName)           // 1. 更新UI状态
  onChange(presetConfig)          // 2. 通知父组件更新config
}

// config变化 → 更新preset
useEffect(() => {
  setPreset(config.style)         // 同步UI状态
}, [config])
```

### 3. 容错处理
- 检查`config?.style`是否存在
- 检查`config?.pullback_ratio !== undefined`
- 提供默认值`setPreset('swing')`

## 验证步骤

1. **创建策略**：选择"剥头皮"风格
2. **保存策略**：点击保存按钮
3. **重新打开**：选择同一策略
4. **验证结果**：剥头皮按钮应该高亮显示

## 相关文件

- `nofx/web/src/components/strategy/TriggerPriceEditor.tsx` - 修复后的组件
- `nofx/web/src/pages/StrategyStudioPage.tsx` - 集成点

## 总结

这是一个典型的React状态同步问题。通过添加`useEffect`，我们确保了：
- ✅ UI状态与配置数据保持一致
- ✅ 用户操作得到正确反馈
- ✅ 页面重新加载后状态正确恢复
