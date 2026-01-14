import React, { useState, useEffect, useCallback } from 'react';
import { Card, Button, InputNumber, message, Spin, Row, Col, Statistic, Tag, Switch, Tooltip } from 'antd';
import { DragOutlined, SaveOutlined, ReloadOutlined, LineChartOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as ChartTooltip, Legend, ResponsiveContainer, ReferenceLine } from 'recharts';
import { api } from '../lib/api';
import type { Position } from '../types';

interface TPSLEditorProps {
  position: Position;
  traderId: string;
  onUpdate?: () => void;
}

interface PriceData {
  time: string;
  price: number;
  timestamp: number;
}

const TPSLEditor: React.FC<TPSLEditorProps> = ({ position, traderId, onUpdate }) => {
  const [currentTP, setCurrentTP] = useState<number>(0);
  const [currentSL, setCurrentSL] = useState<number>(0);
  const [dragging, setDragging] = useState<'tp' | 'sl' | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [priceData, setPriceData] = useState<PriceData[]>([]);
  const [currentPrice, setCurrentPrice] = useState<number>(0);
  const [enableDrag, setEnableDrag] = useState(true);
  const [hasChanges, setHasChanges] = useState(false);

  // 初始化TP/SL值
  useEffect(() => {
    if (position) {
      setCurrentTP(position.take_profit || 0);
      setCurrentSL(position.stop_loss || 0);
      setCurrentPrice(position.current_price || position.entry_price || 0);
      fetchPriceHistory();
    }
  }, [position]);

  // 获取价格历史数据（模拟）
  const fetchPriceHistory = async () => {
    setLoading(true);
    try {
      // 生成模拟价格数据
      const entryPrice = position.entry_price || 0;
      const mockData: PriceData[] = [];
      const now = Date.now();
      
      for (let i = 20; i >= 0; i--) {
        const timestamp = now - i * 60000; // 每分钟一个点
        const randomChange = (Math.random() - 0.5) * entryPrice * 0.01; // ±1%波动
        mockData.push({
          time: new Date(timestamp).toLocaleTimeString(),
          price: entryPrice + randomChange,
          timestamp,
        });
      }
      
      setPriceData(mockData);
      if (mockData.length > 0) {
        setCurrentPrice(mockData[mockData.length - 1].price);
      }
    } catch (error) {
      console.error('Failed to fetch price history:', error);
    } finally {
      setLoading(false);
    }
  };

  // 处理图表拖拽
  const handleChartMouseMove = useCallback(
    (e: any) => {
      if (!dragging || !enableDrag) return;

      const { chartY } = e;
      if (!chartY) return;

      const newPrice = chartY;

      if (dragging === 'tp') {
        setCurrentTP(Number(newPrice.toFixed(2)));
        setHasChanges(true);
      } else if (dragging === 'sl') {
        setCurrentSL(Number(newPrice.toFixed(2)));
        setHasChanges(true);
      }
    },
    [dragging, enableDrag]
  );

  // 保存TP/SL修改
  const handleSave = async () => {
    if (!hasChanges) {
      message.info('没有修改需要保存');
      return;
    }

    setSaving(true);
    try {
      await api.post(`/traders/${traderId}/modify-tpsl`, {
        position_id: position.id,
        new_tp: currentTP,
        new_sl: currentSL,
      });

      message.success('TP/SL 已更新');
      setHasChanges(false);
      if (onUpdate) onUpdate();
    } catch (error) {
      message.error('保存失败: ' + (error instanceof Error ? error.message : '未知错误'));
    } finally {
      setSaving(false);
    }
  };

  // 重置为原始值
  const handleReset = () => {
    setCurrentTP(position.take_profit || 0);
    setCurrentSL(position.stop_loss || 0);
    setHasChanges(false);
    message.info('已重置为原始值');
  };

  // 计算盈亏比
  const calculateRiskReward = () => {
    const entryPrice = position.entry_price || 0;
    const isLong = position.side === 'LONG';

    if (entryPrice === 0 || currentTP === 0 || currentSL === 0) return 0;

    const profitDistance = isLong ? currentTP - entryPrice : entryPrice - currentTP;
    const lossDistance = isLong ? entryPrice - currentSL : currentSL - entryPrice;

    if (lossDistance === 0) return 0;
    return profitDistance / lossDistance;
  };

  const riskReward = calculateRiskReward();
  const entryPrice = position.entry_price || 0;
  const isLong = position.side === 'LONG';

  // 计算潜在盈亏
  const potentialProfit = isLong
    ? (currentTP - entryPrice) * (position.quantity || 0)
    : (entryPrice - currentTP) * (position.quantity || 0);
  
  const potentialLoss = isLong
    ? (entryPrice - currentSL) * (position.quantity || 0)
    : (currentSL - entryPrice) * (position.quantity || 0);

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span>
            <LineChartOutlined /> TP/SL 可视化编辑器
          </span>
          <div>
            <Tooltip title="启用后可以拖拽图表上的线来修改TP/SL">
              <Switch
                checkedChildren="拖拽"
                unCheckedChildren="锁定"
                checked={enableDrag}
                onChange={setEnableDrag}
                style={{ marginRight: 8 }}
              />
            </Tooltip>
            {hasChanges && <Tag color="orange">未保存</Tag>}
          </div>
        </div>
      }
      extra={
        <div>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleReset}
            disabled={!hasChanges}
            style={{ marginRight: 8 }}
          >
            重置
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            onClick={handleSave}
            loading={saving}
            disabled={!hasChanges}
          >
            保存
          </Button>
        </div>
      }
    >
      <Spin spinning={loading}>
        {/* 统计信息 */}
        <Row gutter={16} style={{ marginBottom: 20 }}>
          <Col span={6}>
            <Statistic
              title="持仓方向"
              value={position.side}
              prefix={isLong ? '📈' : '📉'}
              valueStyle={{ color: isLong ? '#52c41a' : '#f5222d' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="入场价格"
              value={entryPrice.toFixed(2)}
              precision={2}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="当前价格"
              value={currentPrice.toFixed(2)}
              precision={2}
              valueStyle={{ color: currentPrice > entryPrice ? '#52c41a' : '#f5222d' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="盈亏比"
              value={riskReward.toFixed(2)}
              precision={2}
              suffix=":1"
              valueStyle={{ color: riskReward >= 2 ? '#52c41a' : riskReward >= 1.5 ? '#faad14' : '#f5222d' }}
            />
          </Col>
        </Row>

        {/* 价格图表 */}
        <div style={{ marginBottom: 20 }}>
          <ResponsiveContainer width="100%" height={400}>
            <LineChart
              data={priceData}
              onMouseMove={handleChartMouseMove}
              onMouseUp={() => setDragging(null)}
              onMouseLeave={() => setDragging(null)}
            >
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis domain={['auto', 'auto']} />
              <ChartTooltip />
              <Legend />
              
              {/* 价格线 */}
              <Line
                type="monotone"
                dataKey="price"
                stroke="#1890ff"
                strokeWidth={2}
                dot={false}
                name="价格"
              />

              {/* 入场价格线 */}
              <ReferenceLine
                y={entryPrice}
                stroke="#722ed1"
                strokeWidth={2}
                strokeDasharray="5 5"
                label={{ value: `入场: ${entryPrice.toFixed(2)}`, position: 'right' }}
              />

              {/* 止盈线 */}
              <ReferenceLine
                y={currentTP}
                stroke="#52c41a"
                strokeWidth={3}
                label={{
                  value: `TP: ${currentTP.toFixed(2)} ${enableDrag ? '(可拖拽)' : ''}`,
                  position: 'right',
                  fill: '#52c41a',
                }}
                style={{ cursor: enableDrag ? 'ns-resize' : 'default' }}
                onMouseDown={() => enableDrag && setDragging('tp')}
              />

              {/* 止损线 */}
              <ReferenceLine
                y={currentSL}
                stroke="#f5222d"
                strokeWidth={3}
                label={{
                  value: `SL: ${currentSL.toFixed(2)} ${enableDrag ? '(可拖拽)' : ''}`,
                  position: 'right',
                  fill: '#f5222d',
                }}
                style={{ cursor: enableDrag ? 'ns-resize' : 'default' }}
                onMouseDown={() => enableDrag && setDragging('sl')}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* 手动输入 */}
        <Row gutter={16} style={{ marginBottom: 20 }}>
          <Col span={8}>
            <div style={{ marginBottom: 8 }}>
              <label>止盈价格 (TP)</label>
            </div>
            <InputNumber
              value={currentTP}
              onChange={(val) => {
                setCurrentTP(val || 0);
                setHasChanges(true);
              }}
              precision={2}
              step={0.01}
              style={{ width: '100%' }}
              prefix="💚"
            />
          </Col>
          <Col span={8}>
            <div style={{ marginBottom: 8 }}>
              <label>止损价格 (SL)</label>
            </div>
            <InputNumber
              value={currentSL}
              onChange={(val) => {
                setCurrentSL(val || 0);
                setHasChanges(true);
              }}
              precision={2}
              step={0.01}
              style={{ width: '100%' }}
              prefix="❤️"
            />
          </Col>
          <Col span={8}>
            <div style={{ marginBottom: 8 }}>
              <label>数量</label>
            </div>
            <InputNumber
              value={position.quantity}
              disabled
              style={{ width: '100%' }}
            />
          </Col>
        </Row>

        {/* 潜在盈亏 */}
        <Row gutter={16}>
          <Col span={12}>
            <Card size="small" style={{ backgroundColor: '#f6ffed', borderColor: '#b7eb8f' }}>
              <Statistic
                title="潜在收益"
                value={potentialProfit}
                precision={2}
                valueStyle={{ color: '#52c41a' }}
                prefix="💰"
                suffix="USDT"
              />
            </Card>
          </Col>
          <Col span={12}>
            <Card size="small" style={{ backgroundColor: '#fff1f0', borderColor: '#ffa39e' }}>
              <Statistic
                title="潜在亏损"
                value={potentialLoss}
                precision={2}
                valueStyle={{ color: '#f5222d' }}
                prefix="⚠️"
                suffix="USDT"
              />
            </Card>
          </Col>
        </Row>

        {/* 提示信息 */}
        {enableDrag && (
          <div style={{ marginTop: 16, padding: 12, backgroundColor: '#e6f7ff', borderRadius: 4 }}>
            <CheckCircleOutlined style={{ color: '#1890ff', marginRight: 8 }} />
            <span style={{ color: '#1890ff' }}>
              拖拽模式已启用：在图表上点击并拖动绿色(TP)或红色(SL)线来调整价格
            </span>
          </div>
        )}
      </Spin>
    </Card>
  );
};

export default TPSLEditor;
