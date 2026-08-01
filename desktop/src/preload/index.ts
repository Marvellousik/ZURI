import { contextBridge, ipcRenderer } from 'electron';
import { IPC_CHANNELS } from '../shared/ipc-channels';
import { DaemonCommandResult, DaemonInfo, IZuriAPI, LogLevel, AppInfo } from '../shared/types';

const zuriAPIBridge: IZuriAPI = {
  getDaemonStatus: (): Promise<DaemonInfo> => {
    return ipcRenderer.invoke(IPC_CHANNELS.DAEMON_GET_STATUS);
  },

  startDaemon: (): Promise<DaemonCommandResult> => {
    return ipcRenderer.invoke(IPC_CHANNELS.DAEMON_START);
  },

  stopDaemon: (): Promise<DaemonCommandResult> => {
    return ipcRenderer.invoke(IPC_CHANNELS.DAEMON_STOP);
  },

  restartDaemon: (): Promise<DaemonCommandResult> => {
    return ipcRenderer.invoke(IPC_CHANNELS.DAEMON_RESTART);
  },

  onDaemonStatusChanged: (callback: (info: DaemonInfo) => void): (() => void) => {
    const handler = (_: unknown, info: DaemonInfo) => callback(info);
    ipcRenderer.on(IPC_CHANNELS.DAEMON_STATUS_CHANGED, handler);
    return () => {
      ipcRenderer.removeListener(IPC_CHANNELS.DAEMON_STATUS_CHANGED, handler);
    };
  },

  logMessage: (level: LogLevel, message: string, details?: unknown): Promise<void> => {
    return ipcRenderer.invoke(IPC_CHANNELS.LOG_MESSAGE, { level, message, details });
  },

  getAppInfo: (): Promise<AppInfo> => {
    return ipcRenderer.invoke(IPC_CHANNELS.APP_GET_INFO);
  },
};

contextBridge.exposeInMainWorld('zuriAPI', zuriAPIBridge);
