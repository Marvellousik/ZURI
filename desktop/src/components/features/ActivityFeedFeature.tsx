import React from 'react';
import { ProvenanceThread, ProvenanceThreadEvent } from '../primitives/ProvenanceThread';
import { Text } from '../primitives/Text';
import { Icon } from '../primitives/Icon';
import { Tag } from '../primitives/Tag';
import { Activity, RefreshCw } from 'lucide-react';
import { Button } from '../primitives/Button';

export interface ActivityFeedFeatureProps {
  events?: ProvenanceThreadEvent[];
  onRefresh?: () => void;
}

const sampleEvents: ProvenanceThreadEvent[] = [
  {
    id: 'evt-104',
    timestamp: '16:42:01',
    description: 'Decision #DEC-892 citation revived during PR review',
    status: 'revived',
    isRevival: true,
    metadata: 'commit: 7f8a91c • target: main',
  },
  {
    id: 'evt-103',
    timestamp: '16:38:45',
    description: 'Postgres vector store sync completed successfully',
    status: 'resolved',
    metadata: '142 memory vectors indexed',
  },
  {
    id: 'evt-102',
    timestamp: '16:30:12',
    description: 'Daemon MCP server ping: health check OK',
    status: 'info',
    metadata: 'latency: 1.2ms',
  },
  {
    id: 'evt-101',
    timestamp: '16:15:00',
    description: 'Workspace ZURI connected to daemon engine',
    status: 'resolved',
    metadata: 'daemon PID: 14208',
  },
];

export const ActivityFeedFeature: React.FC<ActivityFeedFeatureProps> = ({
  events = sampleEvents,
  onRefresh,
}) => {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        padding: '16px',
        gap: '16px',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid var(--color-border)',
          paddingBottom: '12px',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Icon icon={Activity} size={18} style={{ color: 'var(--color-primary)' }} />
          <Text variant="heading" style={{ fontSize: '16px' }}>
            Live Activity
          </Text>
          <Tag>LIVE</Tag>
        </div>
        {onRefresh && (
          <Button
            variant="secondary"
            size="sm"
            onClick={onRefresh}
            icon={<Icon icon={RefreshCw} size={14} />}
            title="Refresh Activity"
          >
            Refresh
          </Button>
        )}
      </div>

      <Text variant="caption" color="secondary">
        Real-time decision provenance thread & daemon audit log stream
      </Text>

      {/* Provenance Thread Timeline Stream */}
      <div style={{ flex: 1, overflowY: 'auto', paddingTop: '8px' }}>
        <ProvenanceThread events={events} />
      </div>
    </div>
  );
};
