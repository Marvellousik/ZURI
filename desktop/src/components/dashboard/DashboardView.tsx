import React, { useState } from 'react';
import { ConnectedRepository, RepoCreateInput } from '../../repositories/repository-repository';
import { useRepositories } from '../../hooks/useRepositories';
import { DaemonOverviewCard } from './DaemonOverviewCard';
import { RepoTable } from './RepoTable';
import { RepoFormModal } from './RepoFormModal';
import { RepoDeleteModal } from './RepoDeleteModal';
import { Button } from '../primitives/Button';
import { Text } from '../primitives/Text';
import { Icon } from '../primitives/Icon';
import { Tag } from '../primitives/Tag';
import { EmptyState } from '../common/EmptyState';
import { AlertBanner } from '../common/AlertBanner';
import { Plus, FolderPlus, Loader2, RefreshCw } from 'lucide-react';

export const DashboardView: React.FC = () => {
  const {
    repositories,
    isLoading,
    error,
    addRepository,
    updateRepository,
    removeRepository,
    refreshRepository,
    validateRepository,
    refreshList,
    clearError,
  } = useRepositories();

  const [isFormModalOpen, setIsFormModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [selectedRepo, setSelectedRepo] = useState<ConnectedRepository | null>(null);
  const [isRefreshingId, setIsRefreshingId] = useState<string | null>(null);
  const [successBanner, setSuccessBanner] = useState<string | null>(null);

  const handleOpenAddModal = () => {
    setSelectedRepo(null);
    setIsFormModalOpen(true);
  };

  const handleOpenEditModal = (repo: ConnectedRepository) => {
    setSelectedRepo(repo);
    setIsFormModalOpen(true);
  };

  const handleOpenDeleteModal = (repo: ConnectedRepository) => {
    setSelectedRepo(repo);
    setIsDeleteModalOpen(true);
  };

  const handleFormSubmit = async (input: RepoCreateInput): Promise<boolean> => {
    if (selectedRepo) {
      const result = await updateRepository(selectedRepo.id, input);
      if (result) {
        setSuccessBanner(`Repository "${result.name}" updated successfully.`);
        return true;
      }
    } else {
      const result = await addRepository(input);
      if (result) {
        setSuccessBanner(`Repository "${result.name}" connected successfully.`);
        return true;
      }
    }
    return false;
  };

  const handleDeleteConfirm = async (id: string): Promise<boolean> => {
    const repoName = selectedRepo?.name || 'Repository';
    const success = await removeRepository(id);
    if (success) {
      setSuccessBanner(`"${repoName}" has been disconnected.`);
      return true;
    }
    return false;
  };

  const handleRefreshSingle = async (repo: ConnectedRepository) => {
    setIsRefreshingId(repo.id);
    await refreshRepository(repo.id);
    setIsRefreshingId(null);
    setSuccessBanner(`Refreshed memory status for "${repo.name}".`);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Overview Cards Section */}
      <DaemonOverviewCard />

      {/* Action Notification Banners */}
      {successBanner && (
        <AlertBanner
          type="success"
          message={successBanner}
          onDismiss={() => setSuccessBanner(null)}
        />
      )}

      {error && (
        <AlertBanner
          type="error"
          message={error}
          onDismiss={clearError}
        />
      )}

      {/* Repositories Section Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '16px',
          flexWrap: 'wrap',
        }}
      >
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Text variant="heading">Connected Repositories</Text>
            <Tag>{repositories.length} Active</Tag>
          </div>
          <Text variant="body" color="secondary" style={{ marginTop: '4px', display: 'block' }}>
            Managed software projects actively serving memory context to AI coding agents
          </Text>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Button
            variant="secondary"
            size="md"
            icon={<Icon icon={RefreshCw} size={14} />}
            onClick={() => refreshList()}
            title="Refresh List"
          >
            Refresh All
          </Button>

          <Button
            variant="primary"
            size="md"
            icon={<Icon icon={Plus} size={16} />}
            onClick={handleOpenAddModal}
          >
            Connect Repository
          </Button>
        </div>
      </div>

      {/* Repository Content Area (Loading, Empty, or Table) */}
      {isLoading ? (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '60px',
            gap: '10px',
          }}
        >
          <Icon icon={Loader2} className="spin" size={20} style={{ color: 'var(--color-primary)' }} />
          <Text variant="data" color="muted">
            Loading connected repositories...
          </Text>
        </div>
      ) : repositories.length === 0 ? (
        <EmptyState
          icon={<Icon icon={FolderPlus} size={32} />}
          title="No Repositories Connected Yet"
          description="Connect a local software repository to start extracting and indexing architectural memory decisions for your AI coding agents."
          action={
            <Button variant="primary" icon={<Icon icon={Plus} size={16} />} onClick={handleOpenAddModal}>
              Connect First Repository
            </Button>
          }
        />
      ) : (
        <RepoTable
          repositories={repositories}
          onRefresh={handleRefreshSingle}
          onEdit={handleOpenEditModal}
          onRemove={handleOpenDeleteModal}
          isRefreshingId={isRefreshingId}
        />
      )}

      {/* Form Dialog Modal */}
      <RepoFormModal
        isOpen={isFormModalOpen}
        onClose={() => setIsFormModalOpen(false)}
        onSubmit={handleFormSubmit}
        onValidate={validateRepository}
        initialData={selectedRepo}
      />

      {/* Delete Confirmation Modal */}
      <RepoDeleteModal
        isOpen={isDeleteModalOpen}
        onClose={() => setIsDeleteModalOpen(false)}
        onConfirm={handleDeleteConfirm}
        repo={selectedRepo}
      />
    </div>
  );
};
