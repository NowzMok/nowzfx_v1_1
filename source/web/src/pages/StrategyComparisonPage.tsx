import React, { useState, useEffect } from 'react';
import { Card, Select, Button, Table, Spin, Empty, message, Row, Col, Tag } from 'antd';
import { BarChart, Bar, LineChart, Line, RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { LineChartOutlined, ReloadOutlined } from '@ant-design/icons';
import { api } from '../lib/api';
import type { TraderInfo } from '../types';

const { Option } = Select;

interface StrategyMetrics {
  strategy_name: string;
  trader_id: string;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_profit: number;
  total_loss: number;
  net_profit: number;
  profit_factor: number;
  max_drawdown: number;
  sharpe_ratio: number;
  average_trade: number;
  average_win: number;
  average_loss: number;
  largest_win: number;
  largest_loss: number;
  trade_duration: number;
  start_date: string;
  end_date: string;
  days_active: number;
  return_rate: number;
  annualized_return: number;
  consecutive_wins: number;
  consecutive_losses: number;
}

interface PerformanceTrend {
  date: string;
  cumulative_roi: number;
  daily_return: number;
  trade_count: number;
  win_rate: number;
}

const COLORS = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1'];

const StrategyComparisonPage: React.FC = () => {
  const [traders, setTraders] = useState<TraderInfo[]>([]);
  const [selectedTraders, setSelectedTraders] = useState<string[]>([]);
  const [comparisons, setComparisons] = useState<StrategyMetrics[]>([]);
  const [trends, setTrends] = useState<Record<string, PerformanceTrend[]>>({});
  const [loading, setLoading] = useState(false);
  const [loadingTraders, setLoadingTraders] = useState(false);

  useEffect(() => {
    fetchTraders();
  }, []);

  useEffect(() => {
    if (selectedTraders.length > 0) {
      fetchComparisonData();
    }
  }, [selectedTraders]);

  const fetchTraders = async () => {
    setLoadingTraders(true);
    try {
      const response = await api.getTraders();
      setTraders(response);
    } catch (error) {
      message.error('获取交易者列表失败');
    } finally {
      setLoadingTraders(false);
    }
  };

  const fetchComparisonData = async () => {
    if (selectedTraders.length === 0) return;

    setLoading(true);
    try {
      const queryParams = selectedTraders.map(id => `trader_ids=${id}`).join('&');
      
      const [compRes, trendRes] = await Promise.all([
        api.get(`/strategy-comparison?${queryParams}`),
        api.get(`/strategy-performance-trend?${queryParams}`),
      ]);

      setComparisons(compRes.comparisons || []);
      setTrends(trendRes.trends || {});
    } catch (err) {
      message.error('获取对比数据失败: ' + (err instanceof Error ? err.message : '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '策略名称',
      dataIndex: 'strategy_name',
      key: 'strategy_name',
      fixed: 'left' as const,
      width: 150,
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: '总交易',
      dataIndex: 'total_trades',
      key: 'total_trades',
      width: 80,
      align: 'center' as const,
    },
    {
      title: '胜率',
      dataIndex: 'win_rate',
      key: 'win_rate',
      width: 100,
      render: (val: number) => (
        <Tag color={val >= 60 ? 'green' : val >= 50 ? 'orange' : 'red'}>
          {val.toFixed(1)}%
        </Tag>
      ),
    },
    {
      title: '净收益',
      dataIndex: 'net_profit',
      key: 'net_profit',
      width: 120,
      render: (val: number) => (
        <span style={{ color: val > 0 ? '#52c41a' : '#f5222d', fontWeight: 'bold' }}>
          {val > 0 ? '+' : ''}{val.toFixed(2)}
        </span>
      ),
    },
    {
      title: '收益率',
      dataIndex: 'return_rate',
      key: 'return_rate',
      width: 100,
      render: (val: number) => (
        <span style={{ color: val > 0 ? '#52c41a' : '#f5222d' }}>
          {val > 0 ? '+' : ''}{val.toFixed(1)}%
        </span>
      ),
    },
    {
      title: '年化收益',
      dataIndex: 'annualized_return',
      key: 'annualized_return',
      width: 120,
      render: (val: number) => `${val > 0 ? '+' : ''}${val.toFixed(1)}%`,
    },
    {
      title: '利润因子',
      dataIndex: 'profit_factor',
      key: 'profit_factor',
      width: 100,
      render: (val: number) => (
        <Tag color={val >= 2 ? 'green' : val >= 1.5 ? 'orange' : 'red'}>
          {val.toFixed(2)}
        </Tag>
      ),
    },
    {
      title: '夏普比率',
      dataIndex: 'sharpe_ratio',
      key: 'sharpe_ratio',
      width: 100,
      render: (val: number) => val.toFixed(2),
    },
    {
      title: '最大回撤',
      dataIndex: 'max_drawdown',
      key: 'max_drawdown',
      width: 100,
      render: (val: number) => (
        <span style={{ color: val < 10 ? '#52c41a' : val < 20 ? '#faad14' : '#f5222d' }}>
          {val.toFixed(1)}%
        </span>
      ),
    },
    {
      title: '平均交易',
      dataIndex: 'average_trade',
      key: 'average_trade',
      width: 100,
      render: (val: number) => val.toFixed(2),
    },
    {
      title: '持仓时间',
      dataIndex: 'trade_duration',
      key: 'trade_duration',
      width: 100,
      render: (val: number) => `${val.toFixed(1)}h`,
    },
  ];

  // 准备雷达图数据
  const prepareRadarData = () => {
    if (comparisons.length === 0) return [];

    const metrics = ['胜率', '利润因子', '夏普比率', '回撤控制', '年化收益'];
    
    return metrics.map(metric => {
      const dataPoint: any = { metric };
      
      comparisons.forEach(comp => {
        let value = 0;
        switch(metric) {
          case '胜率':
            value = comp.win_rate;
            break;
          case '利润因子':
            value = Math.min(comp.profit_factor * 20, 100);
            break;
          case '夏普比率':
            value = Math.min(Math.abs(comp.sharpe_ratio) * 20, 100);
            break;
          case '回撤控制':
            value = Math.max(100 - comp.max_drawdown * 2, 0);
            break;
          case '年化收益':
            value = Math.min(comp.annualized_return, 100);
            break;
        }
        dataPoint[comp.trader_id] = Math.max(0, value);
      });
      
      return dataPoint;
    });
  };

  // 计算排名
  const calculateRanking = () => {
    if (comparisons.length === 0) return [];

    return comparisons
      .map(comp => ({
        ...comp,
        score: (
          comp.win_rate * 0.2 +
          comp.profit_factor * 15 +
          comp.sharpe_ratio * 10 +
          (100 - comp.max_drawdown) * 0.5 +
          comp.return_rate * 0.3
        ),
      }))
      .sort((a, b) => b.score - a.score)
      .map((comp, index) => ({
        rank: index + 1,
        strategy_name: comp.strategy_name,
        score: comp.score.toFixed(1),
        net_profit: comp.net_profit,
        win_rate: comp.win_rate,
        return_rate: comp.return_rate,
      }));
  };

  const rankingColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 80,
      render: (rank: number) => (
        <span style={{ fontSize: '20px', fontWeight: 'bold' }}>
          {rank === 1 ? '🥇' : rank === 2 ? '🥈' : rank === 3 ? '🥉' : `#${rank}`}
        </span>
      ),
    },
    {
      title: '策略名称',
      dataIndex: 'strategy_name',
      key: 'strategy_name',
    },
    {
      title: '综合评分',
      dataIndex: 'score',
      key: 'score',
      render: (score: string) => <Tag color="blue">{score}</Tag>,
    },
    {
      title: '净收益',
      dataIndex: 'net_profit',
      key: 'net_profit',
      render: (val: number) => (
        <span style={{ color: val > 0 ? '#52c41a' : '#f5222d' }}>
          {val > 0 ? '+' : ''}{val.toFixed(2)}
        </span>
      ),
    },
    {
      title: '胜率',
      dataIndex: 'win_rate',
      key: 'win_rate',
      render: (val: number) => `${val.toFixed(1)}%`,
    },
    {
      title: '收益率',
      dataIndex: 'return_rate',
      key: 'return_rate',
      render: (val: number) => `${val > 0 ? '+' : ''}${val.toFixed(1)}%`,
    },
  ];

  const radarData = prepareRadarData();
  const rankedStrategies = calculateRanking();

  // 准备趋势图数据
  const prepareTrendData = () => {
    if (Object.keys(trends).length === 0) return [];
    
    const allDates = new Set<string>();
    Object.values(trends).forEach(trendList => {
      trendList.forEach(t => allDates.add(t.date));
    });
    
    const sortedDates = Array.from(allDates).sort();
    
    return sortedDates.map(date => {
      const dataPoint: any = { date };
      Object.entries(trends).forEach(([traderId, trendList]) => {
        const trend = trendList.find(t => t.date === date);
        dataPoint[traderId] = trend ? trend.cumulative_roi : 0;
      });
      return dataPoint;
    });
  };

  const trendData = prepareTrendData();

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title={
          <span>
            <LineChartOutlined style={{ marginRight: 8 }} />
            多策略性能对比分析
          </span>
        }
        extra={
          <Button icon={<ReloadOutlined />} onClick={fetchComparisonData} loading={loading}>
            刷新
          </Button>
        }
      >
        <Row gutter={16} style={{ marginBottom: 20 }}>
          <Col span={24}>
            <div style={{ marginBottom: 8 }}>
              <label style={{ fontWeight: 500 }}>选择策略 (最多5个):</label>
            </div>
            <Select
              mode="multiple"
              value={selectedTraders}
              onChange={setSelectedTraders}
              style={{ width: '100%' }}
              placeholder="选择要对比的策略"
              maxTagCount={5}
              loading={loadingTraders}
              disabled={selectedTraders.length >= 5}
            >
              {traders.map((trader) => (
                <Option key={trader.trader_id} value={trader.trader_id}>
                  {trader.trader_name || trader.trader_id}
                </Option>
              ))}
            </Select>
          </Col>
        </Row>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <Spin size="large" tip="加载对比数据中..." />
          </div>
        ) : comparisons.length === 0 ? (
          <Empty description="请选择至少一个策略进行对比" style={{ padding: '60px 0' }} />
        ) : (
          <>
            <Card title="📊 策略性能对比" style={{ marginBottom: 20 }} size="small">
              <Table
                dataSource={comparisons}
                columns={columns}
                pagination={false}
                scroll={{ x: 1400 }}
                size="small"
                rowKey="trader_id"
              />
            </Card>

            <Row gutter={16} style={{ marginBottom: 20 }}>
              <Col xs={24} lg={12}>
                <Card title="📈 累计收益率趋势" size="small" style={{ height: '100%' }}>
                  {trendData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={350}>
                      <LineChart data={trendData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="date" />
                        <YAxis />
                        <Tooltip />
                        <Legend />
                        {comparisons.map((comp, index) => (
                          <Line
                            key={comp.trader_id}
                            type="monotone"
                            dataKey={comp.trader_id}
                            stroke={COLORS[index % COLORS.length]}
                            name={comp.strategy_name}
                            strokeWidth={2}
                            dot={false}
                          />
                        ))}
                      </LineChart>
                    </ResponsiveContainer>
                  ) : (
                    <Empty description="无趋势数据" />
                  )}
                </Card>
              </Col>

              <Col xs={24} lg={12}>
                <Card title="🎯 综合指标对比" size="small" style={{ height: '100%' }}>
                  {radarData.length > 0 ? (
                    <ResponsiveContainer width="100%" height={350}>
                      <RadarChart data={radarData}>
                        <PolarGrid />
                        <PolarAngleAxis dataKey="metric" />
                        <PolarRadiusAxis angle={90} domain={[0, 100]} />
                        <Tooltip />
                        <Legend />
                        {comparisons.map((comp, index) => (
                          <Radar
                            key={comp.trader_id}
                            name={comp.strategy_name}
                            dataKey={comp.trader_id}
                            stroke={COLORS[index % COLORS.length]}
                            fill={COLORS[index % COLORS.length]}
                            fillOpacity={0.3}
                          />
                        ))}
                      </RadarChart>
                    </ResponsiveContainer>
                  ) : (
                    <Empty description="无雷达数据" />
                  )}
                </Card>
              </Col>
            </Row>

            <Card title="🏆 综合排名" size="small">
              <Table
                columns={rankingColumns}
                dataSource={rankedStrategies}
                pagination={false}
                size="small"
                rowKey="strategy_name"
              />
            </Card>
          </>
        )}

        <Card title="💡 使用说明" size="small" style={{ marginTop: 20 }}>
          <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
            <li><strong>综合评分</strong>: 基于胜率、利润因子、夏普比率、回撤控制和收益率的加权计算</li>
            <li><strong>雷达图</strong>: 可视化展示各策略在不同维度的表现，面积越大表示综合性能越好</li>
            <li><strong>趋势图</strong>: 展示各策略累计收益率随时间的变化，便于观察稳定性</li>
            <li><strong>最多对比</strong>: 一次最多可选择5个策略进行对比分析</li>
          </ul>
        </Card>
      </Card>

      <style>{`
        .ant-table-tbody tr:nth-child(1) {
          background-color: #fff7e6;
        }
        .ant-table-tbody tr:nth-child(2) {
          background-color: #e6f7ff;
        }
        .ant-table-tbody tr:nth-child(3) {
          background-color: #f6ffed;
        }
      `}</style>
    </div>
  );
};

export default StrategyComparisonPage;
