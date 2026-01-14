# 🎯 前端监控系统集成指南

## 📌 快速开始

### 1. 在现有项目中导入组件

```typescript
import MonitoringDashboard from '@/components/MonitoringDashboard';

export default function TradingPage() {
  return (
    <MonitoringDashboard 
      traderID="your-trader-id" 
      apiBaseURL="http://localhost:8080"
    />
  );
}
```

### 2. 使用 Hook 获取监控数据

```typescript
import { useMonitoring } from '@/hooks/useMonitoring';

function MyComponent() {
  const { metrics, alerts, health, loading, error } = useMonitoring('trader-123');

  if (loading) return <div>加载中...</div>;
  if (error) return <div>错误: {error}</div>;

  return (
    <div>
      <p>胜率: {(metrics[0].win_rate * 100).toFixed(1)}%</p>
      <p>活跃告警: {alerts.length}</p>
    </div>
  );
}
```

## 📊 组件功能

### MonitoringDashboard 组件

完整的一站式监控仪表板，包含：

#### 关键指标卡片
- **胜率** - 历史交易中获利交易的百分比
- **盈利因子** - 总收益/总损失的比率
- **最大回撤** - 从峰值到谷值的最大下降百分比
- **总损益** - 累计收益或损失

#### 性能指标图表
- **胜率趋势** - 显示过去 24 小时的胜率变化
- **损益和回撤** - 面积图展示累计损益趋势
- **回撤趋势** - 柱状图显示回撤百分比

#### 告警管理
- 实时告警列表
- 告警严重级别指示（critical/warning/info）
- 告警状态跟踪（triggered/acknowledged/resolved）
- 快速操作按钮（确认、解决）

#### 系统健康
- 连接状态检查：交易所、数据库、API
- 性能指标：API 延迟、数据库延迟
- 资源使用：内存和 CPU 使用率
- 整体健康评估：健康/降级/不健康

## 🔌 API 集成

### 获取性能指标

```typescript
// 获取最新指标
const getLatestMetric = async (traderID: string) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/metrics/latest`
  );
  return response.json();
};

// 获取多个指标
const getMetrics = async (traderID: string, limit: number = 100) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/metrics?limit=${limit}`
  );
  return response.json();
};

// 获取性能趋势
const getMetricsTrend = async (traderID: string, hours: number = 24) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/metrics/trend?hours=${hours}`
  );
  return response.json();
};
```

### 提交性能数据

```typescript
const collectMetrics = async (traderID: string, metrics: {
  win_rate: number;
  profit_factor: number;
  total_pnl: number;
  daily_pnl: number;
  max_drawdown: number;
  current_drawdown: number;
  sharpe_ratio: number;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  open_positions: number;
  total_equity: number;
  available_balance: number;
  volatility_multiplier: number;
  confidence_adjustment: number;
}) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/metrics/collect`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(metrics),
    }
  );
  return response.json();
};
```

### 告警管理

```typescript
// 获取活跃告警
const getActiveAlerts = async (traderID: string) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/alerts/active`
  );
  return response.json();
};

// 确认告警
const acknowledgeAlert = async (alertID: string) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/alerts/${alertID}/acknowledge`,
    { method: 'POST' }
  );
  return response.json();
};

// 解决告警
const resolveAlert = async (alertID: string) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/alerts/${alertID}/resolve`,
    { method: 'POST' }
  );
  return response.json();
};

// 创建告警规则
const createAlertRule = async (traderID: string, rule: {
  name: string;
  description: string;
  metric_type: string;
  operator: '>' | '<' | '>=' | '<=' | '==';
  threshold: number;
  duration: number;
  severity: 'info' | 'warning' | 'critical';
}) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/alert-rules`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(rule),
    }
  );
  return response.json();
};
```

### 系统健康检查

```typescript
// 获取当前健康状态
const getHealthStatus = async (traderID: string) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/health`
  );
  return response.json();
};

// 执行健康检查
const performHealthCheck = async (traderID: string, healthData: {
  exchange_connected: boolean;
  database_connected: boolean;
  api_healthy: boolean;
  api_latency_ms: number;
  database_latency_ms: number;
  memory_usage_mb: number;
  cpu_usage_percent: number;
}) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/health/check`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(healthData),
    }
  );
  return response.json();
};
```

## 🎨 自定义样式

### Tailwind CSS 集成

组件使用 Tailwind CSS，确保你的项目中已安装并配置了 Tailwind：

```bash
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

### 主题自定义

```typescript
interface MonitoringDashboardProps {
  traderID: string;
  apiBaseURL?: string;
  theme?: 'light' | 'dark';
  refreshInterval?: number; // 毫秒，默认 30000
}
```

## 📈 数据更新流程

### 实时更新

```typescript
// 启用实时数据推送（WebSocket）
const ws = new WebSocket(
  `ws://localhost:8080/api/monitoring/${traderID}/stream`
);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // 更新前端数据
  setMetrics(prev => [data, ...prev]);
};
```

### 定期轮询

```typescript
// 默认每 30 秒刷新一次
// 可通过传入 refreshInterval 属性自定义
<MonitoringDashboard 
  traderID="trader-123"
  refreshInterval={60000} // 60 秒刷新一次
/>
```

## 🔔 告警通知

### 浏览器通知

```typescript
// 当有新告警时显示浏览器通知
if ('Notification' in window) {
  Notification.requestPermission();
  
  const showNotification = (alert: Alert) => {
    new Notification('交易告警', {
      body: alert.message,
      icon: '/alert-icon.png',
      tag: alert.id,
    });
  };
}
```

### 音频提醒

```typescript
const playAlertSound = (severity: 'critical' | 'warning' | 'info') => {
  const audio = new Audio(`/sounds/${severity}-alert.mp3`);
  audio.play();
};
```

## 📊 数据可视化

### 支持的图表类型

- **LineChart** - 趋势展示
- **AreaChart** - 累计数据
- **BarChart** - 对比数据
- **PieChart** - 占比分布

### 自定义图表

```typescript
import { LineChart, Line, ResponsiveContainer } from 'recharts';

const CustomMetricsChart = ({ data }) => (
  <ResponsiveContainer width="100%" height={300}>
    <LineChart data={data}>
      <Line 
        dataKey="win_rate" 
        stroke="#10b981" 
        strokeWidth={2}
      />
    </LineChart>
  </ResponsiveContainer>
);
```

## 🧪 测试

### 模拟数据

```typescript
const mockMetrics: PerformanceMetric[] = [
  {
    id: '1',
    trader_id: 'trader-123',
    timestamp: new Date().toISOString(),
    win_rate: 0.65,
    profit_factor: 2.5,
    total_pnl: 5000,
    max_drawdown: 0.15,
    current_drawdown: 0.05,
    sharpe_ratio: 1.8,
    total_trades: 100,
    winning_trades: 65,
    losing_trades: 35,
    open_positions: 5,
    total_equity: 15000,
  },
];

// 用于组件测试
<MonitoringDashboard traderID="test-trader" />
```

## 🚀 高级功能

### 告警规则编辑器

```typescript
const AlertRuleEditor = ({ traderID }: { traderID: string }) => {
  const [rules, setRules] = useState<AlertRule[]>([]);
  
  const handleCreateRule = async (rule: CreateAlertRuleRequest) => {
    const response = await fetch(
      `http://localhost:8080/api/monitoring/${traderID}/alert-rules`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(rule),
      }
    );
    // 更新规则列表
  };

  return (
    <form onSubmit={(e) => {
      e.preventDefault();
      // 处理表单提交
    }}>
      {/* 规则编辑表单 */}
    </form>
  );
};
```

### 导出报告

```typescript
const exportMetrics = (metrics: PerformanceMetric[], format: 'csv' | 'pdf') => {
  if (format === 'csv') {
    const csv = [
      ['时间', '胜率', '盈利因子', '回撤', '损益'],
      ...metrics.map(m => [
        new Date(m.timestamp).toLocaleString(),
        (m.win_rate * 100).toFixed(1) + '%',
        m.profit_factor.toFixed(2),
        (m.max_drawdown * 100).toFixed(1) + '%',
        m.total_pnl.toFixed(2),
      ])
    ].map(row => row.join(',')).join('\n');
    
    downloadFile(csv, 'metrics.csv');
  }
};
```

## 🔐 安全性

### 认证和授权

```typescript
const fetchMonitoringData = async (traderID: string, token: string) => {
  const response = await fetch(
    `http://localhost:8080/api/monitoring/${traderID}/metrics/latest`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      }
    }
  );
  return response.json();
};
```

### 数据加密

```typescript
// 前端加密敏感字段
import crypto from 'crypto';

const encryptMetrics = (metrics: PerformanceMetric, key: string) => {
  // 实现加密逻辑
};
```

## 📞 常见问题

### Q: 如何实时更新仪表板？
A: 使用 WebSocket 连接或设置适当的 `refreshInterval`。

### Q: 如何自定义告警规则？
A: 使用 `/api/monitoring/{traderID}/alert-rules` 端点创建、更新或删除规则。

### Q: 如何导出监控数据？
A: 调用 `/api/monitoring/{traderID}/metrics` 获取数据，然后导出为 CSV 或 PDF。

### Q: 系统最多能处理多少个交易员？
A: 每个交易员有独立的监控实例，无理论上限，但取决于服务器资源。

## 📚 相关文档

- [监控系统后端文档](./MONITORING_SYSTEM.md)
- [反思系统前端指南](./REFLECTION_FRONTEND_GUIDE.md)
- [API 参考](./API_REFERENCE.md)

---

**最后更新**: 2024-01-12  
**维护者**: AI Trading System Team  
