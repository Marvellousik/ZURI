import React from 'react';

export type TextVariant =
  | 'display'
  | 'heading'
  | 'body'
  | 'body.medium'
  | 'caption'
  | 'data';

export interface TextProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: TextVariant;
  color?: 'primary' | 'secondary' | 'muted' | 'success' | 'warning' | 'danger' | 'info' | 'inherit';
  as?: React.ElementType;
  children: React.ReactNode;
}

export const Text: React.FC<TextProps> = ({
  variant = 'body',
  color = 'primary',
  as,
  children,
  style,
  className,
  ...props
}) => {
  const getVariantStyles = (): React.CSSProperties => {
    switch (variant) {
      case 'display':
        return {
          fontFamily: 'var(--font-sans)',
          fontSize: '28px',
          lineHeight: '34px',
          fontWeight: 600,
        };
      case 'heading':
        return {
          fontFamily: 'var(--font-sans)',
          fontSize: '18px',
          lineHeight: '24px',
          fontWeight: 600,
        };
      case 'body.medium':
        return {
          fontFamily: 'var(--font-sans)',
          fontSize: '14px',
          lineHeight: '20px',
          fontWeight: 500,
        };
      case 'caption':
        return {
          fontFamily: 'var(--font-sans)',
          fontSize: '12px',
          lineHeight: '16px',
          fontWeight: 400,
        };
      case 'data':
        return {
          fontFamily: 'var(--font-mono)',
          fontSize: '13px',
          lineHeight: '18px',
          fontWeight: 400,
        };
      case 'body':
      default:
        return {
          fontFamily: 'var(--font-sans)',
          fontSize: '14px',
          lineHeight: '20px',
          fontWeight: 400,
        };
    }
  };

  const getColorStyles = (): React.CSSProperties => {
    switch (color) {
      case 'secondary':
        return { color: 'var(--color-text-secondary)' };
      case 'muted':
        return { color: 'var(--color-muted)' };
      case 'success':
        return { color: 'var(--color-success)' };
      case 'warning':
        return { color: 'var(--color-warning)' };
      case 'danger':
        return { color: 'var(--color-danger)' };
      case 'info':
        return { color: 'var(--color-info)' };
      case 'inherit':
        return { color: 'inherit' };
      case 'primary':
      default:
        return { color: 'var(--color-text-primary)' };
    }
  };

  const Component = as || (variant === 'display' ? 'h1' : variant === 'heading' ? 'h2' : 'span');

  return (
    <Component
      style={{
        ...getVariantStyles(),
        ...getColorStyles(),
        margin: 0,
        ...style,
      }}
      className={className}
      {...props}
    >
      {children}
    </Component>
  );
};
