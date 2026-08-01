export interface ConnectedRepository {
  id: string;
  name: string;
  localPath: string;
  githubRepoFullName: string;
  defaultBranch: string;
  githubStatus: 'connected' | 'disconnected' | 'pending';
  indexingStatus: 'indexed' | 'indexing' | 'idle' | 'failed';
  lastSyncedAt: string;
  health: 'healthy' | 'warning' | 'error';
  createdAt: string;
}

export type RepoCreateInput = Omit<ConnectedRepository, 'id' | 'createdAt' | 'lastSyncedAt' | 'githubStatus' | 'indexingStatus' | 'health'>;

export interface ValidationResult {
  valid: boolean;
  errors?: Record<string, string>;
}

export interface IRepositoryRepository {
  getAll(): Promise<ConnectedRepository[]>;
  getById(id: string): Promise<ConnectedRepository | null>;
  add(input: RepoCreateInput): Promise<ConnectedRepository>;
  update(id: string, updates: Partial<ConnectedRepository>): Promise<ConnectedRepository>;
  remove(id: string): Promise<void>;
  refresh(id: string): Promise<ConnectedRepository>;
  validate(input: Partial<RepoCreateInput>, currentRepoId?: string): Promise<ValidationResult>;
}

export class RepositoryRepository implements IRepositoryRepository {
  private readonly STORAGE_KEY = 'zuri_connected_repositories';

  private normalizePath(p: string): string {
    return p.trim().toLowerCase().replace(/[/\\]+/g, '/').replace(/\/$/, '');
  }

  public async getAll(): Promise<ConnectedRepository[]> {
    if (typeof window === 'undefined' || !window.localStorage) return [];

    const raw = window.localStorage.getItem(this.STORAGE_KEY);
    if (!raw) {
      // Default initial seed with current workspace project
      const seed: ConnectedRepository[] = [
        {
          id: 'repo-zuri-core',
          name: 'ZURI Core Engine',
          localPath: typeof process !== 'undefined' ? process.cwd() : 'C:/Users/Agada Bartholomew/Documents/ZURI',
          githubRepoFullName: 'Marvellousik/ZURI',
          defaultBranch: 'main',
          githubStatus: 'connected',
          indexingStatus: 'indexed',
          lastSyncedAt: new Date().toISOString(),
          health: 'healthy',
          createdAt: new Date().toISOString(),
        },
      ];
      window.localStorage.setItem(this.STORAGE_KEY, JSON.stringify(seed));
      return seed;
    }

    try {
      return JSON.parse(raw) as ConnectedRepository[];
    } catch {
      return [];
    }
  }

  public async getById(id: string): Promise<ConnectedRepository | null> {
    const all = await this.getAll();
    return all.find((r) => r.id === id) || null;
  }

  public async validate(input: Partial<RepoCreateInput>, currentRepoId?: string): Promise<ValidationResult> {
    const errors: Record<string, string> = {};

    if (!input.name || !input.name.trim()) {
      errors.name = 'Repository name is required';
    }

    if (!input.localPath || !input.localPath.trim()) {
      errors.localPath = 'Local filesystem path is required';
    } else {
      const normalized = this.normalizePath(input.localPath);
      const all = await this.getAll();
      const duplicate = all.find(
        (r) => r.id !== currentRepoId && this.normalizePath(r.localPath) === normalized
      );
      if (duplicate) {
        errors.localPath = `A repository at path "${input.localPath}" is already connected (${duplicate.name})`;
      }
    }

    if (input.githubRepoFullName && input.githubRepoFullName.trim()) {
      const parts = input.githubRepoFullName.trim().split('/');
      if (parts.length !== 2 || !parts[0] || !parts[1]) {
        errors.githubRepoFullName = 'GitHub repository must be in owner/repository format (e.g. org/repo)';
      }
    }

    return {
      valid: Object.keys(errors).length === 0,
      errors: Object.keys(errors).length > 0 ? errors : undefined,
    };
  }

  public async add(input: RepoCreateInput): Promise<ConnectedRepository> {
    const validation = await this.validate(input);
    if (!validation.valid && validation.errors) {
      const firstErr = Object.values(validation.errors)[0];
      throw new Error(firstErr);
    }

    const all = await this.getAll();
    const newRepo: ConnectedRepository = {
      id: `repo-${Math.random().toString(36).substring(2, 9)}`,
      name: input.name.trim(),
      localPath: input.localPath.trim(),
      githubRepoFullName: input.githubRepoFullName ? input.githubRepoFullName.trim() : '',
      defaultBranch: input.defaultBranch ? input.defaultBranch.trim() : 'main',
      githubStatus: input.githubRepoFullName ? 'connected' : 'disconnected',
      indexingStatus: 'idle',
      lastSyncedAt: new Date().toISOString(),
      health: 'healthy',
      createdAt: new Date().toISOString(),
    };

    all.push(newRepo);
    this.save(all);
    return newRepo;
  }

  public async update(id: string, updates: Partial<ConnectedRepository>): Promise<ConnectedRepository> {
    const all = await this.getAll();
    const index = all.findIndex((r) => r.id === id);
    if (index === -1) throw new Error(`Repository with ID ${id} not found`);

    const current = all[index];
    const validation = await this.validate(
      {
        name: updates.name ?? current.name,
        localPath: updates.localPath ?? current.localPath,
        githubRepoFullName: updates.githubRepoFullName ?? current.githubRepoFullName,
        defaultBranch: updates.defaultBranch ?? current.defaultBranch,
      },
      id
    );

    if (!validation.valid && validation.errors) {
      const firstErr = Object.values(validation.errors)[0];
      throw new Error(firstErr);
    }

    const updated: ConnectedRepository = {
      ...current,
      ...updates,
      id: current.id,
      createdAt: current.createdAt,
    };

    all[index] = updated;
    this.save(all);
    return updated;
  }

  public async remove(id: string): Promise<void> {
    const all = await this.getAll();
    const filtered = all.filter((r) => r.id !== id);
    this.save(filtered);
  }

  public async refresh(id: string): Promise<ConnectedRepository> {
    const all = await this.getAll();
    const index = all.findIndex((r) => r.id === id);
    if (index === -1) throw new Error(`Repository with ID ${id} not found`);

    const repo = all[index];
    const refreshed: ConnectedRepository = {
      ...repo,
      lastSyncedAt: new Date().toISOString(),
      indexingStatus: 'indexed',
      health: 'healthy',
    };

    all[index] = refreshed;
    this.save(all);
    return refreshed;
  }

  private save(repos: ConnectedRepository[]): void {
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem(this.STORAGE_KEY, JSON.stringify(repos));
    }
  }
}

export const repositoryRepository = new RepositoryRepository();
