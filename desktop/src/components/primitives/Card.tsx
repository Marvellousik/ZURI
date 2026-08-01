import React from 'react';
import { Text } from './Text';

export interface CardProps {
  children: React.ReactNode;
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  action?: React.ReactNode;
  style?: React.CSSProperties;
  className?: string;
}

export const Card: React.FC<CardProps> = ({
  children,
  title,
  subtitle,
  action,
  style,
  className,
}) => {
  return (
    <div
      style={{
        backgroundColor: 'var(--color-surface)',
        border: '1px solid var(--color-border)',
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-level-1)',
        padding: '16px',
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
        ...style,
      }}
      className={className}
    >
      {(title || action) && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px' }}>
          <div>
            {typeof title === 'string' ? (
              <Text variant="heading">{title}</Text>
            ) : (
              title
            )}
            {subtitle && (
              <div style={{ marginTop: '2px' }}>
                {typeof subtitle === 'string' ? (
                  <Text variant="caption" color="secondary">
                    {subtitle}
                  </Text>
                ) : (
                  subtitle
                )}
              </div>
            )}
          </div>
          {action && <div>{action}</div>}
        </div>
      )}
      <div>{children}</div>
    </div>
  );
};
