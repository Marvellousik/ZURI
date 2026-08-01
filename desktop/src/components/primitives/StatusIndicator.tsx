import React from 'react';
import { Text } from './Text';

export type StatusVariant = 'success' | 'warning' | 'danger' | 'info' | 'muted';

export interface StatusIndicatorProps {
  status: StatusVariant;
  label: string;
  details?: string;
  style?: React.CSSProperties;
  className?: string;
}

export const StatusIndicator: React.FC<StatusIndicatorProps> = ({
  status,
  label,
  details,
  style,
  className,
}) => {
  const getDotColor = (): string => {
    switch (status) {
      case 'success':
        return 'var(--color-success)';
      case 'warning':
        return 'var(--color-warning)';
      case 'danger':
        return 'var(--color-danger)';
      case 'info':
        return 'var(--color-info)';
      case 'muted':
      default:
        return 'var(--color-muted)';
    }
  };

  return (
    <div
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        ...style,
      }}
      className={className}
    >
      <span
        style={{
          width: '6px',
          height: '6px',
          borderRadius: '50%',
          backgroundColor: getDotColor(),
          display: 'inline-block',
          flexShrink: 0,
        }}
      />
      <Text variant="data" style={{ fontWeight: 500, letterSpacing: '0.04em' }}>
        {label.toUpperCase()}
      </Text>
      {details && (
        <Text variant="data" color="muted">
          {details}
        </Text>
      )}
    </div>
  );
};
