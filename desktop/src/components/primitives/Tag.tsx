import React from 'react';
import { Text } from './Text';

export interface TagProps {
  children: React.ReactNode;
  style?: React.CSSProperties;
  className?: string;
}

export const Tag: React.FC<TagProps> = ({ children, style, className }) => {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        padding: '2px 8px',
        borderRadius: 'var(--radius-pill)',
        backgroundColor: 'var(--color-background)',
        border: '1px solid var(--color-border)',
        ...style,
      }}
      className={className}
    >
      {typeof children === 'string' ? (
        <Text variant="caption" color="muted">
          {children}
        </Text>
      ) : (
        children
      )}
    </span>
  );
};
