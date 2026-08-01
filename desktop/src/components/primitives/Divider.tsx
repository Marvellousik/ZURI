import React from 'react';

export interface DividerProps {
  orientation?: 'horizontal' | 'vertical';
  style?: React.CSSProperties;
  className?: string;
}

export const Divider: React.FC<DividerProps> = ({
  orientation = 'horizontal',
  style,
  className,
}) => {
  return (
    <div
      style={{
        backgroundColor: 'var(--color-border)',
        width: orientation === 'horizontal' ? '100%' : '1px',
        height: orientation === 'horizontal' ? '1px' : '100%',
        flexShrink: 0,
        ...style,
      }}
      className={className}
    />
  );
};
