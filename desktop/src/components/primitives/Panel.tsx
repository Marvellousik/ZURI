import React from 'react';

export interface PanelProps {
  children: React.ReactNode;
  variant?: 'sidebar' | 'main' | 'rail' | 'container';
  width?: string;
  style?: React.CSSProperties;
  className?: string;
}

export const Panel: React.FC<PanelProps> = ({
  children,
  variant = 'container',
  width,
  style,
  className,
}) => {
  const getPanelStyles = (): React.CSSProperties => {
    switch (variant) {
      case 'sidebar':
        return {
          width: width || '240px',
          flexShrink: 0,
          backgroundColor: 'var(--color-background)',
          borderRight: '1px solid var(--color-border)',
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        };
      case 'rail':
        return {
          width: width || '320px',
          flexShrink: 0,
          backgroundColor: 'var(--color-surface)',
          borderLeft: '1px solid var(--color-border)',
          boxShadow: 'var(--shadow-level-1)',
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          overflowY: 'auto',
        };
      case 'main':
        return {
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          overflowY: 'auto',
          backgroundColor: 'var(--color-background)',
        };
      case 'container':
      default:
        return {
          backgroundColor: 'var(--color-surface)',
          border: '1px solid var(--color-border)',
          borderRadius: 'var(--radius-md)',
          padding: '16px',
        };
    }
  };

  return (
    <div
      style={{
        ...getPanelStyles(),
        ...style,
      }}
      className={className}
    >
      {children}
    </div>
  );
};
