import { DaemonCommandResult, DaemonInfo } from '../shared/types';
import { logger } from './logger-service';

export interface IDaemonService {
  getStatus(): Promise<DaemonInfo>;
  start(): Promise<DaemonCommandResult>;
  stop(): Promise<DaemonCommandResult>;
  restart(): Promise<DaemonCommandResult>;
  onStatusChanged(callback: (info: DaemonInfo) => void): () => void;
}

class DaemonService implements IDaemonService {
  public async getStatus(): Promise<DaemonInfo> {
    if (typeof window !== 'undefined' && window.zuriAPI?.getDaemonStatus) {
      return await window.zuriAPI.getDaemonStatus();
    }
    // Fallback for isolated browser/test environments
    return {
      status: 'STOPPED',
      port: 7331,
      host: '127.0.0.1',
      lastCheckedAt: new Date().toISOString(),
      error: 'IPC zuriAPI not available in window context',
    };
  }

  public async start(): Promise<DaemonCommandResult> {
    logger.info('Requesting daemon start...');
    if (typeof window !== 'undefined' && window.zuriAPI?.startDaemon) {
      const res = await window.zuriAPI.startDaemon();
      if (!res.success) {
        logger.error(`Daemon start failed: ${res.error}`);
      } else {
        logger.info('Daemon started successfully.');
      }
      return res;
    }
    return { success: false, error: 'IPC zuriAPI unavailable' };
  }

  public async stop(): Promise<DaemonCommandResult> {
    logger.info('Requesting daemon stop...');
    if (typeof window !== 'undefined' && window.zuriAPI?.stopDaemon) {
      const res = await window.zuriAPI.stopDaemon();
      if (!res.success) {
        logger.error(`Daemon stop failed: ${res.error}`);
      } else {
        logger.info('Daemon stopped successfully.');
      }
      return res;
    }
    return { success: false, error: 'IPC zuriAPI unavailable' };
  }

  public async restart(): Promise<DaemonCommandResult> {
    logger.info('Requesting daemon restart...');
    if (typeof window !== 'undefined' && window.zuriAPI?.restartDaemon) {
      const res = await window.zuriAPI.restartDaemon();
      if (!res.success) {
        logger.error(`Daemon restart failed: ${res.error}`);
      } else {
        logger.info('Daemon restarted successfully.');
      }
      return res;
    }
    return { success: false, error: 'IPC zuriAPI unavailable' };
  }

  public onStatusChanged(callback: (info: DaemonInfo) => void): () => void {
    if (typeof window !== 'undefined' && window.zuriAPI?.onDaemonStatusChanged) {
      return window.zuriAPI.onDaemonStatusChanged(callback);
    }
    return () => {};
  }
}

export const daemonService = new DaemonService();
