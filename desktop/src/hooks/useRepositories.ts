import { useState, useEffect, useCallback } from 'react';
import {
  ConnectedRepository,
  RepoCreateInput,
  repositoryRepository,
  IRepositoryRepository,
  ValidationResult,
} from '../repositories/repository-repository';
import { logger } from '../services/logger-service';

export function useRepositories(repoInstance: IRepositoryRepository = repositoryRepository) {
  const [repositories, setRepositories] = useState<ConnectedRepository[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRepositories = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await repoInstance.getAll();
      setRepositories(data);
    } catch (err: any) {
      const msg = err.message || 'Failed to load repositories';
      setError(msg);
      logger.error('useRepositories fetch failed', { error: err });
    } finally {
      setIsLoading(false);
    }
  }, [repoInstance]);

  useEffect(() => {
    fetchRepositories();
  }, [fetchRepositories]);

  const addRepository = async (input: RepoCreateInput): Promise<ConnectedRepository | null> => {
    setError(null);
    try {
      const created = await repoInstance.add(input);
      logger.info(`Connected repository added: ${created.name}`);
      await fetchRepositories();
      return created;
    } catch (err: any) {
      const msg = err.message || 'Failed to add repository';
      setError(msg);
      return null;
    }
  };

  const updateRepository = async (id: string, updates: Partial<ConnectedRepository>): Promise<ConnectedRepository | null> => {
    setError(null);
    try {
      const updated = await repoInstance.update(id, updates);
      logger.info(`Connected repository updated: ${updated.name}`);
      await fetchRepositories();
      return updated;
    } catch (err: any) {
      const msg = err.message || 'Failed to update repository';
      setError(msg);
      return null;
    }
  };

  const removeRepository = async (id: string): Promise<boolean> => {
    setError(null);
    try {
      await repoInstance.remove(id);
      logger.info(`Connected repository removed ID: ${id}`);
      await fetchRepositories();
      return true;
    } catch (err: any) {
      const msg = err.message || 'Failed to remove repository';
      setError(msg);
      return false;
    }
  };

  const refreshRepository = async (id: string): Promise<boolean> => {
    setError(null);
    try {
      await repoInstance.refresh(id);
      logger.info(`Refreshed repository ID: ${id}`);
      await fetchRepositories();
      return true;
    } catch (err: any) {
      const msg = err.message || 'Failed to refresh repository';
      setError(msg);
      return false;
    }
  };

  const validateRepository = async (input: Partial<RepoCreateInput>, currentRepoId?: string): Promise<ValidationResult> => {
    return repoInstance.validate(input, currentRepoId);
  };

  return {
    repositories,
    isLoading,
    error,
    addRepository,
    updateRepository,
    removeRepository,
    refreshRepository,
    validateRepository,
    refreshList: fetchRepositories,
    clearError: () => setError(null),
  };
}
