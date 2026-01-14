import React, { useState, useEffect } from 'react';
import { BarChart, Bar, LineChart, Line, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Card, Row, Col, Statistic, Spin, Table, Empty, Tag, Tabs } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { api } from '../lib/api';

interface OrderStatistics {
  total_orders: number;
  executed_orders: number;
  cancelled_orders: number;
  pending_orders: number;
  execution_rate: number;
  success_rate: number;
  average_wait_time: number;
  total_profit: number;
  total_loss: number;
  profit_factor: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  average_profit_per_trade: number;
  max_drawdown: number;
  time_range: string;
}

interface OrderTrend {
  date: string;
  order_count: number;
  executed_count: number;
  success_rate: number;
  daily_profit: number;
}

interface SymbolStats {
  symbol: string;
  count: number;
  executed: number;
  cancelled: number;
  total_profit: number;
  win_rate: number;
  winning: number;
  losing: number;
}

interface OrderStatisticsPanelProps {
  traderId?: string;
}

const OrderStatisticsPanel: React.FC<OrderStatisticsPanelProps> = ({ traderId = 'default_trader' }) => {
  const [statistics, setStatistics] = useState<OrderStatistics | null>(null);
  const [trends, setTrends] = useState<OrderTrend[]>([]);
  const [symbolStats, setSymbolStats] = useState<{ [key: string]: SymbolStats }>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchData();
    // 每30秒刷新一次数据
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [traderId]);

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [statsRes, trendsRes, symbolRes] = await Promise.all([
        api.getOrderStatistics(traderId),
        api.getOrderStatisticsTrend(traderId),
        api.getOrdersBySymbol(traderId),
      ]);

      setStatistics(statsRes);
      setTrends(trendsRes?.trends || []);
      setSymbolStats(symbolRes?.statistics || {});
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch data');
      console.error('Error fetching order statistics:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading && !statistics) {
    return <Spin tip="加载订单统计中..." />;
  }

  if (error && !statistics) {
    return <Card type="inner" title="订单统计面板" style={{ color: 'red' }}>错误: {error}</Card>;
  }

  const stats = statistics!;

  // 生成图表数据
  const profitChartData = [
    { name: '赢利', value: stats.total_profit, fill: '#52c41a' },
    { name: '亏损', value: -stats.total_loss, fill: '#f5222d' },
  ].filter(item => item.value > 0);

  const symbolTableData = Object.values(symbolStats).map((s: SymbolStats) => ({
    key: s.symbol,
    symbol: s.symbol,
    count: s.count,
    executed: s.executed,
    cancelled: s.cancelled,
    total_profit: s.total_profit.toFixed(4),
    win_rate: s.win_rate.toFixed(2),
    winning: s.winning,
    losing: s.losing,
  }));

  const symbolColumns = [
    { title: '币种', dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: '总订单', dataIndex: 'count', key: 'count', width: 80, align: 'center' as const },
    { title: '已执行', dataIndex: 'executed', key: 'executed', width: 80, align: 'center' as const },
    { title: '已取消', dataIndex: 'cancelled', key: 'cancelled', width: 80, align: 'center' as const },
    {
      title: '总利润',
      dataIndex: 'total_profit',
      key: 'total_profit',
      width: 100,
      render: (text: string) => {
        const value = parseFloat(text);
        return (
          <span style={{ color: value > 0 ? '#52c41a' : '#f5222d' }}>
            {value > 0 ? '+' : ''}{text}
          </span>
        );
      },
    },
    { title: '胜率', dataIndex: 'win_rate', key: 'win_rate', width: 80, render: (text: string) => `${text}%` },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Card title="📊 订单分析统计面板" extra={<span style={{ fontSize: '12px', color: '#999' }}>{stats.time_range}</span>}>
        <Tabs
          items={[
            {
              key: '1',
              label: '概览',
              children: (
                <div>
                  <Row gutter={16} style={{ marginBottom: '24px' }}>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="总订单"
                        value={stats.total_orders}
                        prefix="📦"
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="已执行"
                        value={stats.executed_orders}
                        prefix="✅"
                        suffix={`/ ${stats.total_orders}`}
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="执行率"
                        value={stats.execution_rate}
                        precision={2}
                        suffix="%"
                        valueStyle={{ color: stats.execution_rate >= 80 ? '#52c41a' : '#faad14' }}
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="成功率"
                        value={stats.success_rate}
                        precision={2}
                        suffix="%"
                        valueStyle={{ color: stats.success_rate >= 85 ? '#52c41a' : '#faad14' }}
                      />
                    </Col>
                  </Row>

                  <Row gutter={16} style={{ marginBottom: '24px' }}>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="总利润"
                        value={stats.total_profit}
                        precision={4}
                        valueStyle={{ color: stats.total_profit > 0 ? '#52c41a' : '#f5222d' }}
                        prefix={stats.total_profit > 0 ? <ArrowUpOutlined /> : <ArrowDownOutlined />}
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="总亏损"
                        value={-stats.total_loss}
                        precision={4}
                        valueStyle={{ color: '#f5222d' }}
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="利润因子"
                        value={stats.profit_factor}
                        precision={2}
                        valueStyle={{ color: stats.profit_factor > 1.5 ? '#52c41a' : '#faad14' }}
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="胜率"
                        value={stats.win_rate}
                        precision={2}
                        suffix="%"
                        valueStyle={{ color: stats.win_rate >= 50 ? '#52c41a' : '#f5222d' }}
                      />
                    </Col>
                  </Row>

                  <Row gutter={16} style={{ marginBottom: '24px' }}>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="赢利交易"
                        value={stats.winning_trades}
                        prefix="🟢"
                        suffix={`/ ${stats.winning_trades + stats.losing_trades}`}
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="亏损交易"
                        value={stats.losing_trades}
                        prefix="🔴"
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="平均等待"
                        value={Math.round(stats.average_wait_time)}
                        suffix="秒"
                      />
                    </Col>
                    <Col xs={12} sm={12} md={6}>
                      <Statistic
                        title="最大回撤"
                        value={stats.max_drawdown}
                        precision={2}
                        suffix="%"
                        valueStyle={{ color: stats.max_drawdown < 10 ? '#52c41a' : '#faad14' }}
                      />
                    </Col>
                  </Row>

                  {/* 利润分布饼图 */}
                  <Card title="💰 利润分布" style={{ marginTop: '20px' }} size="small">
                    {profitChartData.length > 0 ? (
                      <ResponsiveContainer width="100%" height={300}>
                        <PieChart>
                          <Pie
                            data={profitChartData}
                            cx="50%"
                            cy="50%"
                            labelLine={false}
                            label={({ name, value }) => `${name}: ${value.toFixed(4)}`}
                            outerRadius={100}
                            fill="#8884d8"
                            dataKey="value"
                          >
                            {profitChartData.map((entry, index) => (
                              <Cell key={`cell-${index}`} fill={entry.fill} />
                            ))}
                          </Pie>
                          <Tooltip />
                        </PieChart>
                      </ResponsiveContainer>
                    ) : (
                      <Empty description="无交易数据" />
                    )}
                  </Card>
                </div>
              ),
            },
            {
              key: '2',
              label: '趋势',
              children: (
                <div>
                  {trends.length > 0 ? (
                    <>
                      <Card title="📈 每日交易数量" size="small" style={{ marginBottom: '20px' }}>
                        <ResponsiveContainer width="100%" height={300}>
                          <BarChart data={trends}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="date" />
                            <YAxis />
                            <Tooltip />
                            <Legend />
                            <Bar dataKey="order_count" fill="#8884d8" name="总订单" />
                            <Bar dataKey="executed_count" fill="#82ca9d" name="已执行" />
                          </BarChart>
                        </ResponsiveContainer>
                      </Card>

                      <Card title="💹 每日利润趋势" size="small">
                        <ResponsiveContainer width="100%" height={300}>
                          <LineChart data={trends}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="date" />
                            <YAxis />
                            <Tooltip />
                            <Legend />
                            <Line
                              type="monotone"
                              dataKey="daily_profit"
                              stroke="#f5222d"
                              strokeWidth={2}
                              name="日利润"
                              dot={{ r: 4 }}
                            />
                            <Line
                              type="monotone"
                              dataKey="success_rate"
                              stroke="#52c41a"
                              strokeWidth={2}
                              name="成功率%"
                              dot={{ r: 4 }}
                            />
                          </LineChart>
                        </ResponsiveContainer>
                      </Card>
                    </>
                  ) : (
                    <Empty description="无趋势数据" />
                  )}
                </div>
              ),
            },
            {
              key: '3',
              label: '币种分析',
              children: (
                <div>
                  {symbolTableData.length > 0 ? (
                    <Table
                      columns={symbolColumns}
                      dataSource={symbolTableData}
                      pagination={{ pageSize: 10 }}
                      size="small"
                      scroll={{ x: 600 }}
                    />
                  ) : (
                    <Empty description="无币种数据" />
                  )}
                </div>
              ),
            },
          ]}
        />

        <div style={{ marginTop: '20px', textAlign: 'center' }}>
          <button
            onClick={fetchData}
            disabled={loading}
            style={{
              padding: '8px 16px',
              backgroundColor: '#1890ff',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            {loading ? '刷新中...' : '刷新数据'}
          </button>
        </div>
      </Card>
    </div>
  );
};

export default OrderStatisticsPanel;
