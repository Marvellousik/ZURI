import React, { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Icon } from './Icon';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'link' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  isLoading?: boolean;
  icon?: React.ReactNode;
  children: React.ReactNode;
}

export const Button: React.FC<ButtonProps> = ({
  variant = 'primary',
  size = 'md',
  isLoading = false,
  icon,
  children,
  disabled,
  style,
  onMouseEnter,
  onMouseLeave,
  ...props
}) => {
  const [isHovered, setIsHovered] = useState(false);

  const getBaseStyles = (): React.CSSProperties => {
    switch (variant) {
      case 'secondary':
        return {
          backgroundColor: isHovered ? 'var(--color-primary-tint)' : 'var(--color-surface)',
          color: 'var(--color-text-primary)',
          border: '1px solid var(--color-border)',
        };
      case 'link':
        return {
          backgroundColor: 'transparent',
          color: isHovered ? 'var(--color-primary-hover)' : 'var(--color-primary)',
          border: 'none',
          padding: 0,
          textDecoration: isHovered ? 'underline' : 'none',
        };
      case 'danger':
        return {
          backgroundColor: isHovered ? '#722F29' : 'var(--color-danger)',
          color: '#FFFFFF',
          border: 'none',
        };
      case 'primary':
      default:
        return {
          backgroundColor: isHovered ? 'var(--color-primary-hover)' : 'var(--color-primary)',
          color: 'var(--color-on-primary)',
          border: 'none',
        };
    }
  };

  const getSizeStyles = (): React.CSSProperties => {
    if (variant === 'link') {
      return { fontSize: '14px', lineHeight: '20px' };
    }
    switch (size) {
      case 'sm':
        return { padding: '4px 10px', fontSize: '12px', lineHeight: '16px' };
      case 'lg':
        return { padding: '10px 20px', fontSize: '14px', lineHeight: '20px' };
      case 'md':
      default:
        return { padding: '6px 14px', fontSize: '14px', lineHeight: '20px' };
    }
  };

  const isBtnDisabled = disabled || isLoading;

  return (
    <button
      disabled={isBtnDisabled}
      onMouseEnter={(e) => {
        setIsHovered(true);
        onMouseEnter?.(e);
      }}
      onMouseLeave={(e) => {
        setIsHovered(false);
        onMouseLeave?.(e);
      }}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '8px',
        fontFamily: 'var(--font-sans)',
        fontWeight: 500,
        borderRadius: variant === 'link' ? 0 : 'var(--radius-sm)',
        cursor: isBtnDisabled ? 'not-allowed' : 'pointer',
        opacity: isBtnDisabled ? 0.6 : 1,
        transition: 'background-color var(--motion-fast) var(--easing-standard), color var(--motion-fast) var(--easing-standard)',
        boxShadow: 'none',
        ...getBaseStyles(),
        ...getSizeStyles(),
        ...style,
      }}
      {...props}
    >
      {isLoading ? <Icon icon={Loader2} className="spin" size={size === 'sm' ? 14 : 18} /> : icon}
      <span>{children}</span>
    </button>
  );
};
