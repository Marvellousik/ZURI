import React, { useState } from 'react';
import { Text } from './Text';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  helperText?: string;
  error?: string;
}

export const Input: React.FC<InputProps> = ({
  label,
  helperText,
  error,
  id,
  style,
  onFocus,
  onBlur,
  ...props
}) => {
  const [isFocused, setIsFocused] = useState(false);
  const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', width: '100%' }}>
      {label && (
        <label htmlFor={inputId}>
          <Text variant="caption" color="secondary">
            {label}
          </Text>
        </label>
      )}
      <input
        id={inputId}
        onFocus={(e) => {
          setIsFocused(true);
          onFocus?.(e);
        }}
        onBlur={(e) => {
          setIsFocused(false);
          onBlur?.(e);
        }}
        style={{
          width: '100%',
          padding: '8px 12px',
          borderRadius: 'var(--radius-sm)',
          backgroundColor: 'var(--color-surface)',
          border: `1px solid ${
            error
              ? 'var(--color-danger)'
              : isFocused
              ? 'var(--color-border-strong)'
              : 'var(--color-border)'
          }`,
          color: 'var(--color-text-primary)',
          fontFamily: 'var(--font-sans)',
          fontSize: '14px',
          lineHeight: '20px',
          outline: 'none',
          transition: 'border-color var(--motion-fast) var(--easing-standard)',
          ...style,
        }}
        {...props}
      />
      {error && (
        <Text variant="caption" color="danger">
          {error}
        </Text>
      )}
      {!error && helperText && (
        <Text variant="caption" color="muted">
          {helperText}
        </Text>
      )}
    </div>
  );
};
