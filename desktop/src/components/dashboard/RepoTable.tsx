import React from 'react';
import { ConnectedRepository } from '../../repositories/repository-repository';
import { Table, Column } from '../primitives/Table';
import { Button } from '../primitives/Button';
import { Text } from '../primitives/Text';
import { Icon } from '../primitives/Icon';
import { Tag } from '../primitives/Tag';
import { StatusIndicator } from '../primitives/StatusIndicator';
import { Folder, GitBranch, RefreshCw, Edit3, Trash2, GitPullRequest } from 'lucide-react';

export interface RepoTableProps {
  repositories: ConnectedRepository[];
  onRefresh: (repo: ConnectedRepository) => void;
  onEdit: (repo: ConnectedRepository) => void;
  onRemove: (repo: ConnectedRepository) => void;
  isRefreshingId?: string | null;
}

export const RepoTable: React.FC<RepoTableProps> = ({
  repositories,
  onRefresh,
  onEdit,
  onRemove,
  isRefreshingId,
}) => {
  const formatDate = (isoString: string) => {
    try {
      const d = new Date(isoString);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) + ' ' + d.toLocaleDateString();
    } catch {
      return isoString;
    }
  };

  const columns: Column<ConnectedRepository>[] = [
    {
      key: 'name',
      header: 'Repository Name',
      render: (repo) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div
            style={{
              padding: '6px',
              borderRadius: 'var(--radius-sm)',
              backgroundColor: 'var(--color-primary-tint)',
              color: 'var(--color-primary)',
              display: 'flex',
            }}
          >
            <Icon icon={Folder} size={16} />
          </div>
          <div>
            <Text variant="body.medium">{repo.name}</Text>
            <div style={{ display: 'flex', alignItems: 'center', gap: '4px', marginTop: '2px' }}>
              <Icon icon={GitBranch} size={12} style={{ color: 'var(--color-muted)' }} />
              <Text variant="data" color="muted">
                {repo.defaultBranch}
              </Text>
            </div>
          </div>
        </div>
      ),
    },
    {
      key: 'localPath',
      header: 'Local Path',
      render: (repo) => (
        <code
          style={{
            fontSize: '12px',
            fontFamily: 'var(--font-mono)',
            color: 'var(--color-text-secondary)',
            backgroundColor: 'var(--color-background)',
            padding: '2px 6px',
            borderRadius: 'var(--radius-pill)',
            border: '1px solid var(--color-border)',
            wordBreak: 'break-all',
          }}
        >
          {repo.localPath}
        </code>
      ),
    },
    {
      key: 'githubRepoFullName',
      header: 'GitHub Integration',
      render: (repo) =>
        repo.githubRepoFullName ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <Icon icon={GitPullRequest} size={14} style={{ color: 'var(--color-muted)' }} />
            <Text variant="data">{repo.githubRepoFullName}</Text>
          </div>
        ) : (
          <Tag>Unlinked</Tag>
        ),
    },
    {
      key: 'indexingStatus',
      header: 'Indexing Status',
      render: (repo) => {
        const variantMap: Record<string, 'success' | 'warning' | 'danger' | 'muted'> = {
          indexed: 'success',
          indexing: 'warning',
          failed: 'danger',
          idle: 'muted',
          unindexed: 'muted',
        };
        return (
          <StatusIndicator
            status={variantMap[repo.indexingStatus] || 'muted'}
            label={repo.indexingStatus}
          />
        );
      },
    },
    {
      key: 'lastSyncedAt',
      header: 'Last Synced',
      render: (repo) => (
        <Text variant="data" color="muted">
          {formatDate(repo.lastSyncedAt)}
        </Text>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      align: 'right',
      render: (repo) => {
        const isRefreshing = isRefreshingId === repo.id;
        return (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
            <Button
              variant="secondary"
              size="sm"
              icon={<Icon icon={RefreshCw} size={14} className={isRefreshing ? 'spin' : ''} />}
              onClick={(e) => {
                e.stopPropagation();
                onRefresh(repo);
              }}
              title="Sync Memory"
              disabled={isRefreshing}
            >
              Sync
            </Button>
            <Button
              variant="secondary"
              size="sm"
              icon={<Icon icon={Edit3} size={14} />}
              onClick={(e) => {
                e.stopPropagation();
                onEdit(repo);
              }}
              title="Edit Repository Settings"
            >
              Edit
            </Button>
            <Button
              variant="danger"
              size="sm"
              icon={<Icon icon={Trash2} size={14} />}
              onClick={(e) => {
                e.stopPropagation();
                onRemove(repo);
              }}
              title="Disconnect Repository"
            >
              Disconnect
            </Button>
          </div>
        );
      },
    },
  ];

  return (
    <Table
      data={repositories}
      columns={columns}
      keyExtractor={(repo) => repo.id}
    />
  );
};
