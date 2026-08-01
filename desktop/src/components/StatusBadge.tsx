import React from 'react';
import { DaemonStatus } from '../shared/types';
import { StatusIndicator, StatusVariant } from './primitives/StatusIndicator';

interface StatusBadgeProps {
  status: DaemonStatus;
  port?: number;
  showDetails?: boolean;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, port, showDetails = false }) => {
  const getStatusConfig = (s: DaemonStatus): { label: string; variant: StatusVariant } => {
    switch (s) {
      case 'RUNNING':
        return { label: 'RUNNING', variant: 'success' };
      case 'STARTING':
        return { label: 'STARTING', variant: 'warning' };
      case 'STOPPING':
        return { label: 'STOPPING', variant: 'warning' };
      case 'ERROR':
        return { label: 'ERROR', variant: 'danger' };
      case 'STOPPED':
      default:
        return { label: 'OFFLINE', variant: 'muted' };
    }
  };

  const config = getStatusConfig(status);
  const details = showDetails && port && status === 'RUNNING' ? `:${port}` : undefined;

  return <StatusIndicator status={config.variant} label={config.label} details={details} />;
};
