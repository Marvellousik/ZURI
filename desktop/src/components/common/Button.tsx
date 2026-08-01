import React from 'react';
import { Button as PrimitiveButton, ButtonProps as PrimitiveButtonProps } from '../primitives/Button';

export interface ButtonProps extends Omit<PrimitiveButtonProps, 'variant'> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline' | 'link';
}

export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  ...props
}) => {
  // Map legacy/convenience variant names to primitive variants
  const mappedVariant =
    variant === 'ghost' || variant === 'outline' ? 'secondary' : variant;

  return <PrimitiveButton variant={mappedVariant} {...props} />;
};
