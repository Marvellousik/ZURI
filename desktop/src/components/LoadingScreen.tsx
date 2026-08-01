import React from 'react';
import { Brain, Loader2 } from 'lucide-react';
import { Text } from './primitives/Text';
import { Icon } from './primitives/Icon';

export const LoadingScreen: React.FC = () => {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        width: '100vw',
        backgroundColor: 'var(--color-background)',
        color: 'var(--color-text-primary)',
        gap: '24px',
      }}
    >
      <div
        style={{
          width: '48px',
          height: '48px',
          borderRadius: 'var(--radius-md)',
          backgroundColor: 'var(--color-primary-tint)',
          color: 'var(--color-primary)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: '1px solid var(--color-border)',
        }}
      >
        <Icon icon={Brain} size={28} />
      </div>

      <div style={{ textAlign: 'center' }}>
        <Text variant="heading" style={{ display: 'block', marginBottom: '8px' }}>
          ZURI Engine Shell
        </Text>
        <Text variant="body" color="secondary">
          Initializing IPC bridge & detecting daemon engine status
        </Text>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <Icon icon={Loader2} className="spin" size={18} style={{ color: 'var(--color-primary)' }} />
        <Text variant="data" color="muted">
          Loading...
        </Text>
      </div>
    </div>
  );
};
