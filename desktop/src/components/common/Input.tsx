import React from 'react';
import { Input as PrimitiveInput, InputProps as PrimitiveInputProps } from '../primitives/Input';

export type InputProps = PrimitiveInputProps;

export const Input: React.FC<InputProps> = (props) => {
  return <PrimitiveInput {...props} />;
};
