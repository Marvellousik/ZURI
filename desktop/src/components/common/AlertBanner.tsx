import React from 'react';
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react';
import { Text } from '../primitives/Text';
import { Icon } from '../primitives/Icon';

export type AlertType = 'info' | 'success' | 'warning' | 'error';

export interface AlertBannerProps {
  type?: AlertType;
  message: string;
  onDismiss?: () => void;
}

export const AlertBanner: React.FC<AlertBannerProps> = ({ type = 'info', message, onDismiss }) => {
  const getTypeConfig = () => {
    switch (type) {
      case 'success':
        return {
          bg: 'var(--color-surface)',
          border: 'var(--color-success)',
          textColor: 'success' as const,
          icon: CheckCircle2,
        };
      case 'warning':
        return {
          bg: 'var(--color-surface)',
          border: 'var(--color-warning)',
          textColor: 'warning' as const,
          icon: AlertTriangle,
        };
      case 'error':
        return {
          bg: 'var(--color-surface)',
          border: 'var(--color-danger)',
          textColor: 'danger' as const,
          icon: AlertTriangle,
        };
      case 'info':
      default:
        return {
          bg: 'var(--color-surface)',
          border: 'var(--color-info)',
          textColor: 'info' as const,
          icon: Info,
        };
    }
  };

  const config = getTypeConfig();

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '10px 16px',
        borderRadius: 'var(--radius-sm)',
        backgroundColor: config.bg,
        border: '1px solid var(--color-border)',
        borderLeft: `4px solid ${config.border}`,
        gap: '12px',
        boxShadow: 'var(--shadow-level-1)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <Icon icon={config.icon} size={18} style={{ color: config.border }} />
        <Text variant="body" color={config.textColor}>
          {message}
        </Text>
      </div>
      {onDismiss && (
        <button
          onClick={onDismiss}
          aria-label="Dismiss banner"
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--color-muted)',
            cursor: 'pointer',
            padding: '2px',
            borderRadius: 'var(--radius-pill)',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Icon icon={X} size={18} />
        </button>
      )}
    </div>
  );
};
