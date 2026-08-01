import React from 'react';
import { useDaemonStatus } from '../../hooks/useDaemonStatus';
import { Card } from '../primitives/Card';
import { Text } from '../primitives/Text';
import { Icon } from '../primitives/Icon';
import { StatusBadge } from '../StatusBadge';
import { Server, Database, Activity, Cpu, ShieldCheck } from 'lucide-react';

export const DaemonOverviewCard: React.FC = () => {
  const { status, port, host, pid, uptimeSeconds, isRunning, error } = useDaemonStatus();

  const formatUptime = (seconds?: number) => {
    if (!seconds) return 'Offline';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) return `${h}h ${m}m ${s}s`;
    return `${m}m ${s}s`;
  };

  return (
    <Card
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Icon icon={Server} size={18} style={{ color: 'var(--color-primary)' }} />
          <Text variant="heading">Daemon Core Engine Status</Text>
        </div>
      }
      subtitle="Standing local MCP server & embedded Postgres background process"
      action={<StatusBadge status={status} port={port} showDetails={true} />}
    >
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
          gap: '16px',
        }}
      >
        {/* Metric 1: Connection & Port */}
        <div
          style={{
            padding: '12px 16px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-background)',
            border: '1px solid var(--color-border)',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          <Icon icon={Activity} size={18} style={{ color: 'var(--color-primary)' }} />
          <div>
            <Text variant="caption" color="secondary" style={{ display: 'block' }}>
              Connection & Endpoint
            </Text>
            <Text variant="data">
              {isRunning ? `http://${host}:${port}` : 'Disconnected'}
            </Text>
          </div>
        </div>

        {/* Metric 2: Uptime & PID */}
        <div
          style={{
            padding: '12px 16px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-background)',
            border: '1px solid var(--color-border)',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          <Icon icon={Cpu} size={18} style={{ color: 'var(--color-success)' }} />
          <div>
            <Text variant="caption" color="secondary" style={{ display: 'block' }}>
              Engine Uptime & PID
            </Text>
            <Text variant="data">
              {formatUptime(uptimeSeconds)} {pid ? `(PID ${pid})` : ''}
            </Text>
          </div>
        </div>

        {/* Metric 3: Database & Vector Engine */}
        <div
          style={{
            padding: '12px 16px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-background)',
            border: '1px solid var(--color-border)',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          <Icon icon={Database} size={18} style={{ color: 'var(--color-info)' }} />
          <div>
            <Text variant="caption" color="secondary" style={{ display: 'block' }}>
              Embedded Store
            </Text>
            <Text variant="data">
              {isRunning ? 'Postgres 18 + pgvector' : 'Stopped'}
            </Text>
          </div>
        </div>

        {/* Metric 4: Active Background Jobs */}
        <div
          style={{
            padding: '12px 16px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-background)',
            border: '1px solid var(--color-border)',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          <Icon icon={ShieldCheck} size={18} style={{ color: 'var(--color-warning)' }} />
          <div>
            <Text variant="caption" color="secondary" style={{ display: 'block' }}>
              Background Services
            </Text>
            <Text variant="data">
              {isRunning ? 'Scheduler & Webhook Ready' : 'Idle'}
            </Text>
          </div>
        </div>
      </div>

      {error && (
        <div
          style={{
            marginTop: '12px',
            padding: '8px 12px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-surface)',
            border: '1px solid var(--color-border)',
            borderLeft: '4px solid var(--color-danger)',
          }}
        >
          <Text variant="caption" color="danger">
            <strong>Daemon Diagnostic Error:</strong> {error}
          </Text>
        </div>
      )}
    </Card>
  );
};
