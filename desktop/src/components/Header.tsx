import React from 'react';
import { useAppState } from '../state/AppContext';
import { useDaemonStatus } from '../hooks/useDaemonStatus';
import { useTheme } from '../hooks/useTheme';
import { StatusBadge } from './StatusBadge';
import { Brain, Sun, Moon, AlertTriangle, X } from 'lucide-react';
import { Text } from './primitives/Text';
import { Icon } from './primitives/Icon';
import { Button } from './primitives/Button';

export const Header: React.FC = () => {
  const { lastError, clearError } = useAppState();
  const { status } = useDaemonStatus();
  const { theme, toggleTheme } = useTheme();

  return (
    <header
      style={{
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: 'var(--color-surface)',
        borderBottom: '1px solid var(--color-border)',
        zIndex: 100,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 24px',
          height: '56px',
        }}
      >
        {/* Brand / Logo */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div
            style={{
              width: '32px',
              height: '32px',
              borderRadius: 'var(--radius-sm)',
              backgroundColor: 'var(--color-primary-tint)',
              color: 'var(--color-primary)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              border: '1px solid var(--color-border)',
            }}
          >
            <Icon icon={Brain} size={18} />
          </div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Text variant="heading" style={{ fontSize: '16px', letterSpacing: '-0.01em' }}>
                ZURI
              </Text>
              <Text variant="caption" color="muted">
                v1.0
              </Text>
            </div>
            <Text variant="caption" color="secondary" style={{ display: 'block', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              Engineering Decision Audit Trail
            </Text>
          </div>
        </div>

        {/* Header Right Status & Theme */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <StatusBadge status={status} />

          <Button
            variant="secondary"
            size="sm"
            onClick={toggleTheme}
            icon={<Icon icon={theme === 'dark' ? Sun : Moon} size={14} />}
            title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
          >
            {theme === 'dark' ? 'Light' : 'Dark'} Mode
          </Button>
        </div>
      </div>

      {/* Error Banner Notification */}
      {lastError && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '8px 24px',
            backgroundColor: 'var(--color-surface)',
            borderTop: '1px solid var(--color-border)',
            borderBottom: '1px solid var(--color-border)',
            borderLeft: '4px solid var(--color-danger)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Icon icon={AlertTriangle} size={16} style={{ color: 'var(--color-danger)' }} />
            <Text variant="body" color="danger">
              {lastError}
            </Text>
          </div>
          <button
            onClick={clearError}
            aria-label="Dismiss error"
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--color-danger)',
              cursor: 'pointer',
              padding: '2px',
              borderRadius: 'var(--radius-pill)',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            <Icon icon={X} size={14} />
          </button>
        </div>
      )}
    </header>
  );
};
