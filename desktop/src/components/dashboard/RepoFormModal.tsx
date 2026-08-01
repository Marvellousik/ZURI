import React, { useState, useEffect } from 'react';
import { ConnectedRepository, RepoCreateInput, ValidationResult } from '../../repositories/repository-repository';
import { Modal } from '../primitives/Modal';
import { Input } from '../primitives/Input';
import { Button } from '../primitives/Button';

export interface RepoFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (input: RepoCreateInput) => Promise<boolean>;
  onValidate: (input: Partial<RepoCreateInput>, currentRepoId?: string) => Promise<ValidationResult>;
  initialData?: ConnectedRepository | null;
}

export const RepoFormModal: React.FC<RepoFormModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
  onValidate,
  initialData,
}) => {
  const isEditing = Boolean(initialData);

  const [name, setName] = useState('');
  const [localPath, setLocalPath] = useState('');
  const [githubRepoFullName, setGithubRepoFullName] = useState('');
  const [defaultBranch, setDefaultBranch] = useState('main');

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (initialData) {
      setName(initialData.name);
      setLocalPath(initialData.localPath);
      setGithubRepoFullName(initialData.githubRepoFullName || '');
      setDefaultBranch(initialData.defaultBranch || 'main');
    } else {
      setName('');
      setLocalPath('');
      setGithubRepoFullName('');
      setDefaultBranch('main');
    }
    setErrors({});
  }, [initialData, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    const input: RepoCreateInput = {
      name,
      localPath,
      githubRepoFullName,
      defaultBranch,
    };

    const validation = await onValidate(input, initialData?.id);
    if (!validation.valid && validation.errors) {
      setErrors(validation.errors);
      setIsSubmitting(false);
      return;
    }

    const success = await onSubmit(input);
    setIsSubmitting(false);
    if (success) {
      onClose();
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Repository Settings' : 'Connect New Repository'}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSubmit} isLoading={isSubmitting}>
            {isEditing ? 'Save Changes' : 'Connect Repository'}
          </Button>
        </>
      }
    >
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
        <Input
          label="Repository Display Name"
          placeholder="e.g. ZURI Core Engine"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            if (errors.name) setErrors((prev) => ({ ...prev, name: '' }));
          }}
          error={errors.name}
          helperText="A human-readable label for this repository"
        />

        <Input
          label="Local Filesystem Directory Path"
          placeholder="e.g. C:/Users/Developer/Documents/ZURI"
          value={localPath}
          onChange={(e) => {
            setLocalPath(e.target.value);
            if (errors.localPath) setErrors((prev) => ({ ...prev, localPath: '' }));
          }}
          error={errors.localPath}
          helperText="Absolute local path to the workspace repository"
        />

        <Input
          label="GitHub Full Repository Name (Optional)"
          placeholder="e.g. owner/repository-name"
          value={githubRepoFullName}
          onChange={(e) => {
            setGithubRepoFullName(e.target.value);
            if (errors.githubRepoFullName) setErrors((prev) => ({ ...prev, githubRepoFullName: '' }));
          }}
          error={errors.githubRepoFullName}
          helperText="Used for PR webhook memory extraction & resolution workflows"
        />

        <Input
          label="Default Target Branch"
          placeholder="main"
          value={defaultBranch}
          onChange={(e) => setDefaultBranch(e.target.value)}
          helperText="Branch that establishes canonical memory"
        />
      </form>
    </Modal>
  );
};
