import React from 'react';
import { LucideProps } from 'lucide-react';

export interface IconProps extends LucideProps {
  icon: React.ComponentType<LucideProps>;
}

export const Icon: React.FC<IconProps> = ({
  icon: LucideComponent,
  size = 18,
  strokeWidth = 1.75,
  color = 'currentColor',
  className,
  style,
  ...props
}) => {
  return (
    <LucideComponent
      size={size}
      strokeWidth={strokeWidth}
      color={color}
      className={className}
      style={{
        flexShrink: 0,
        ...style,
      }}
      {...props}
    />
  );
};
