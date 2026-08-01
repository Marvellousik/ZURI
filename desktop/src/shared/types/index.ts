/**
 * Shared Type Definitions for Zuri Desktop Application
 */

export type DaemonStatus = 'STOPPED' | 'STARTING' | 'RUNNING' | 'STOPPING' | 'ERROR';

export interface DaemonInfo {
  status: DaemonStatus;
  port: number;
  host: string;
  pid?: number;
  uptimeSeconds?: number;
  error?: string;
  lastCheckedAt: string;
}

export type NavigationTab = 'dashboard' | 'explorer' | 'activity' | 'settings';

export type LogLevel = 'info' | 'warn' | 'error';

export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  message: string;
  source: 'main' | 'renderer' | 'daemon';
  details?: unknown;
}

export interface AppInfo {
  name: string;
  version: string;
  platform: string;
  arch: string;
  isDev: boolean;
}

export interface DaemonCommandResult {
  success: boolean;
  message?: string;
  error?: string;
  daemonInfo?: DaemonInfo;
}

/**
 * Type-safe IPC API interface exposed via Context Bridge (window.zuriAPI)
 */
export interface IZuriAPI {
  getDaemonStatus: () => Promise<DaemonInfo>;
  startDaemon: () => Promise<DaemonCommandResult>;
  stopDaemon: () => Promise<DaemonCommandResult>;
  restartDaemon: () => Promise<DaemonCommandResult>;
  onDaemonStatusChanged: (callback: (status: DaemonInfo) => void) => () => void;
  logMessage: (level: LogLevel, message: string, details?: unknown) => Promise<void>;
  getAppInfo: () => Promise<AppInfo>;
}
