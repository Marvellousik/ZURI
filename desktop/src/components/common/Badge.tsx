import React from 'react';
import { Tag } from '../primitives/Tag';
import { StatusIndicator } from '../primitives/StatusIndicator';

export type BadgeVariant = 'success' | 'warning' | 'danger' | 'neutral' | 'info';

export interface BadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  pulse?: boolean;
  size?: 'sm' | 'md';
}

export const Badge: React.FC<BadgeProps> = ({ children, variant = 'neutral' }) => {
  if (typeof children === 'string') {
    const statusMap: Record<BadgeVariant, 'success' | 'warning' | 'danger' | 'info' | 'muted'> = {
      success: 'success',
      warning: 'warning',
      danger: 'danger',
      info: 'info',
      neutral: 'muted',
    };
    return <StatusIndicator status={statusMap[variant]} label={children} />;
  }

  return <Tag>{children}</Tag>;
};
