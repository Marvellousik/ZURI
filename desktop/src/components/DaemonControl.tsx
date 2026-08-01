import React from 'react';
import { useDaemonStatus } from '../hooks/useDaemonStatus';
import { StatusBadge } from './StatusBadge';
import { Play, Square, RotateCw, Server } from 'lucide-react';
import { Text } from './primitives/Text';
import { Icon } from './primitives/Icon';
import { Button } from './primitives/Button';

export const DaemonControl: React.FC = () => {
  const { status, port, host, pid, uptimeSeconds, isRunning, isStarting, isStopping, start, stop, restart } = useDaemonStatus();

  const isBusy = isStarting || isStopping;

  const formatUptime = (seconds?: number) => {
    if (!seconds) return null;
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}m ${s}s`;
  };

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '16px',
        backgroundColor: 'var(--color-surface)',
        border: '1px solid var(--color-border)',
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-level-1)',
        gap: '16px',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <div
          style={{
            padding: '8px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-primary-tint)',
            color: 'var(--color-primary)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            border: '1px solid var(--color-border)',
          }}
        >
          <Icon icon={Server} size={18} />
        </div>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Text variant="heading" style={{ fontSize: '16px' }}>
              Engine Daemon
            </Text>
            <StatusBadge status={status} port={port} showDetails={true} />
          </div>
          <div style={{ marginTop: '2px', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Text variant="data" color="muted">
              {host}:{port}
            </Text>
            {pid && (
              <Text variant="data" color="muted">
                • PID {pid}
              </Text>
            )}
            {uptimeSeconds && (
              <Text variant="data" color="muted">
                • Uptime {formatUptime(uptimeSeconds)}
              </Text>
            )}
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        {!isRunning ? (
          <Button
            variant="primary"
            size="sm"
            onClick={() => start()}
            disabled={isBusy}
            icon={<Icon icon={Play} size={14} />}
          >
            Start Daemon
          </Button>
        ) : (
          <Button
            variant="danger"
            size="sm"
            onClick={() => stop()}
            disabled={isBusy}
            icon={<Icon icon={Square} size={14} />}
          >
            Stop Daemon
          </Button>
        )}

        <Button
          variant="secondary"
          size="sm"
          onClick={() => restart()}
          disabled={isBusy}
          title="Restart Daemon Process"
          icon={<Icon icon={RotateCw} size={14} className={isBusy ? 'spin' : ''} />}
        >
          Restart
        </Button>
      </div>
    </div>
  );
};
