import React from 'react';
import { Modal as PrimitiveModal, ModalProps as PrimitiveModalProps } from '../primitives/Modal';

export type ModalProps = PrimitiveModalProps;

export const Modal: React.FC<ModalProps> = (props) => {
  return <PrimitiveModal {...props} />;
};
