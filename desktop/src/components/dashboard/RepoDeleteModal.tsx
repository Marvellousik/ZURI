import React, { useState } from 'react';
import { ConnectedRepository } from '../../repositories/repository-repository';
import { Modal } from '../primitives/Modal';
import { Button } from '../primitives/Button';
import { Text } from '../primitives/Text';
import { Icon } from '../primitives/Icon';
import { AlertTriangle } from 'lucide-react';

export interface RepoDeleteModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (id: string) => Promise<boolean>;
  repo: ConnectedRepository | null;
}

export const RepoDeleteModal: React.FC<RepoDeleteModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  repo,
}) => {
  const [isDeleting, setIsDeleting] = useState(false);

  if (!repo) return null;

  const handleDelete = async () => {
    setIsDeleting(true);
    const success = await onConfirm(repo.id);
    setIsDeleting(false);
    if (success) {
      onClose();
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Disconnect Repository"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isDeleting}>
            Cancel
          </Button>
          <Button variant="danger" onClick={handleDelete} isLoading={isDeleting}>
            Confirm Disconnect
          </Button>
        </>
      }
    >
      <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
        <div
          style={{
            padding: '10px',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-surface)',
            color: 'var(--color-danger)',
            border: '1px solid var(--color-border)',
            borderLeft: '4px solid var(--color-danger)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          <Icon icon={AlertTriangle} size={20} />
        </div>
        <div>
          <Text variant="heading" style={{ fontSize: '16px', marginBottom: '6px', display: 'block' }}>
            Are you sure you want to disconnect "{repo.name}"?
          </Text>
          <Text variant="body" color="secondary">
            This removes the repository configuration from the Zuri daemon UI view. No actual memory records or local git files will be deleted.
          </Text>
        </div>
      </div>
    </Modal>
  );
};
