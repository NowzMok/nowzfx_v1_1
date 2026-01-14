import React, { useState, useEffect } from 'react';
import { Card, Select, Spin, Empty, message, Button, Row, Col } from 'antd';
import { ReloadOutlined, LineChartOutlined } from '@ant-design/icons';
import TPSLEditor from '../components/TPSLEditor';
import { api } from '../lib/api';
import type { Position } from '../types';

const { Option } = Select;

interface TPSLManagementPageProps {
  traderId?: string;
}

const TPSLManagementPage: React.FC<TPSLManagementPageProps> = ({ traderId = 'default_trader' }) => {
  const [positions, setPositions] = useState<Position[]>([]);
  const [selectedPosition, setSelectedPosition] = useState<Position | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchPositions();
  }, [traderId]);

  const fetchPositions = async () => {
    setLoading(true);
    try {
      const response = await api.getPositions(traderId);
      const openPositions = response.filter((p: Position) => p.status === 'OPEN');
      setPositions(openPositions);
      
      if (openPositions.length > 0 && !selectedPosition) {
        setSelectedPosition(openPositions[0]);
      }
    } catch (error) {
      message.error('获取持仓失败: ' + (error instanceof Error ? error.message : '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  const handlePositionChange = (positionId: number) => {
    const position = positions.find(p => p.id === positionId);
    if (position) {
      setSelectedPosition(position);
    }
  };

  const handleUpdate = () => {
    fetchPositions();
    message.success('持仓数据已刷新');
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title={
          <span>
            <LineChartOutlined style={{ marginRight: 8 }} />
            TP/SL 可视化管理
          </span>
        }
        extra={
          <Button icon={<ReloadOutlined />} onClick={fetchPositions} loading={loading}>
            刷新
          </Button>
        }
      >
        <Row gutter={16} style={{ marginBottom: 20 }}>
          <Col span={24}>
            <div style={{ marginBottom: 8 }}>
              <label style={{ fontWeight: 500 }}>选择持仓:</label>
            </div>
            <Select
              value={selectedPosition?.id}
              onChange={handlePositionChange}
              style={{ width: '100%' }}
              loading={loading}
              placeholder="请选择一个持仓"
            >
              {positions.map((pos) => (
                <Option key={pos.id} value={pos.id}>
                  {pos.symbol} - {pos.side} - {pos.quantity} @ {pos.entry_price?.toFixed(2)} 
                  {pos.unrealized_pnl !== undefined && (
                    <span style={{ 
                      marginLeft: 8,
                      color: pos.unrealized_pnl > 0 ? '#52c41a' : '#f5222d' 
                    }}>
                      ({pos.unrealized_pnl > 0 ? '+' : ''}{pos.unrealized_pnl.toFixed(2)} USDT)
                    </span>
                  )}
                </Option>
              ))}
            </Select>
          </Col>
        </Row>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <Spin size="large" tip="加载持仓中..." />
          </div>
        ) : selectedPosition ? (
          <TPSLEditor
            position={selectedPosition}
            traderId={traderId}
            onUpdate={handleUpdate}
          />
        ) : (
          <Empty
            description="没有开放的持仓"
            style={{ padding: '60px 0' }}
          >
            <p style={{ color: '#999' }}>当前没有需要管理的持仓</p>
          </Empty>
        )}
      </Card>

      {/* 帮助信息 */}
      <Card 
        title="💡 使用说明" 
        size="small" 
        style={{ marginTop: 20 }}
      >
        <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
          <li>
            <strong>拖拽模式:</strong> 启用后，在图表上直接点击并拖动绿色(TP)或红色(SL)线来调整价格
          </li>
          <li>
            <strong>手动输入:</strong> 也可以在下方输入框中手动输入精确的价格
          </li>
          <li>
            <strong>盈亏比:</strong> 系统会自动计算当前设置的盈亏比，建议保持在2:1以上
          </li>
          <li>
            <strong>实时同步:</strong> 保存后，TP/SL将同步到交易所（如果支持）
          </li>
          <li>
            <strong>价格参考:</strong> 紫色虚线表示入场价格，蓝色实线表示当前价格走势
          </li>
        </ul>
      </Card>
    </div>
  );
};

export default TPSLManagementPage;
