import React from 'react';
import { Text } from './Text';

export interface ProvenanceThreadEvent {
  id: string;
  timestamp: string;
  description: React.ReactNode;
  status?: 'resolved' | 'revived' | 'pending' | 'failed' | 'info';
  isRevival?: boolean;
  lit?: boolean;
  metadata?: string;
}

export interface ProvenanceThreadProps {
  events: ProvenanceThreadEvent[];
  style?: React.CSSProperties;
  className?: string;
}

export const ProvenanceThread: React.FC<ProvenanceThreadProps> = ({
  events,
  style,
  className,
}) => {
  const getStatusColor = (status?: string): string => {
    switch (status) {
      case 'resolved':
        return 'var(--color-success)';
      case 'revived':
        return 'var(--color-primary)';
      case 'pending':
        return 'var(--color-warning)';
      case 'failed':
        return 'var(--color-danger)';
      case 'info':
      default:
        return 'var(--color-info)';
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
        paddingLeft: '20px',
        ...style,
      }}
      className={className}
    >
      {events.map((event, index) => {
        const isLast = index === events.length - 1;
        const isLit = event.lit !== false;
        const isRevival = Boolean(event.isRevival);

        // Segment line color: solid primary if lit or revival; tint if unlit
        const lineColor = isLit || isRevival ? 'var(--color-primary)' : 'var(--color-primary-tint)';
        const dotColor = getStatusColor(event.status);

        return (
          <div
            key={event.id}
            style={{
              position: 'relative',
              paddingBottom: isLast ? '0px' : '16px',
              animation: 'activityFeedEnter var(--motion-normal) var(--easing-standard)',
            }}
          >
            {/* 2px Vertical Provenance Thread Line */}
            {!isLast && (
              <div
                style={{
                  position: 'absolute',
                  left: '-15px',
                  top: '10px',
                  bottom: '0px',
                  width: '2px',
                  backgroundColor: lineColor,
                  transition: 'background-color var(--motion-fast) var(--easing-standard)',
                }}
              />
            )}

            {/* Event Dot on Thread Line */}
            <div
              style={{
                position: 'absolute',
                left: '-18px',
                top: '4px',
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                backgroundColor: dotColor,
                border: isRevival ? '2px solid var(--color-primary)' : 'none',
                boxShadow: isRevival ? '0 0 0 2px var(--color-surface)' : 'none',
              }}
            />

            {/* Event Row Content */}
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '8px', flexWrap: 'wrap' }}>
              <Text variant="data" color="muted" style={{ minWidth: '70px' }}>
                {event.timestamp}
              </Text>
              <div style={{ flex: 1 }}>
                <Text variant="body">{event.description}</Text>
                {event.metadata && (
                  <div style={{ marginTop: '2px' }}>
                    <Text variant="data" color="muted">
                      {event.metadata}
                    </Text>
                  </div>
                )}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};
