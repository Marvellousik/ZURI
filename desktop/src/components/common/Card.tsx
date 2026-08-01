import React from 'react';
import { Card as PrimitiveCard, CardProps as PrimitiveCardProps } from '../primitives/Card';

export type CardProps = PrimitiveCardProps;

export const Card: React.FC<CardProps> = (props) => {
  return <PrimitiveCard {...props} />;
};
