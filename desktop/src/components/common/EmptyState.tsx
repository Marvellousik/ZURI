import React from 'react';
import { Text } from '../primitives/Text';

export interface EmptyStateProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
}

export const EmptyState: React.FC<EmptyStateProps> = ({ icon, title, description, action }) => {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '48px 24px',
        textAlign: 'center',
        gap: '16px',
        backgroundColor: 'var(--color-surface)',
        border: '1px solid var(--color-border)',
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-level-1)',
      }}
    >
      <div
        style={{
          padding: '12px',
          borderRadius: 'var(--radius-md)',
          backgroundColor: 'var(--color-primary-tint)',
          color: 'var(--color-primary)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {icon}
      </div>

      <div style={{ maxWidth: '400px' }}>
        <Text variant="heading" style={{ display: 'block', marginBottom: '6px' }}>
          {title}
        </Text>
        <Text variant="body" color="secondary">
          {description}
        </Text>
      </div>

      {action && <div style={{ marginTop: '8px' }}>{action}</div>}
    </div>
  );
};
