import { ipcMain, BrowserWindow, app } from 'electron';
import { IPC_CHANNELS } from '../shared/ipc-channels';
import { DaemonProcessManager } from '../main/daemon';
import { LogLevel, AppInfo } from '../shared/types';

export function registerIPCHandlers(
  daemonManager: DaemonProcessManager,
  getMainWindow: () => BrowserWindow | null
): void {
  // Daemon lifecycle handlers
  ipcMain.handle(IPC_CHANNELS.DAEMON_GET_STATUS, async () => {
    return daemonManager.getDaemonInfo();
  });

  ipcMain.handle(IPC_CHANNELS.DAEMON_START, async () => {
    return await daemonManager.start();
  });

  ipcMain.handle(IPC_CHANNELS.DAEMON_STOP, async () => {
    return await daemonManager.stop();
  });

  ipcMain.handle(IPC_CHANNELS.DAEMON_RESTART, async () => {
    return await daemonManager.restart();
  });

  // Logging handler
  ipcMain.handle(
    IPC_CHANNELS.LOG_MESSAGE,
    async (_, { level, message, details }: { level: LogLevel; message: string; details?: unknown }) => {
      const timestamp = new Date().toISOString();
      const formatted = `[Renderer Log] [${level.toUpperCase()}] ${message}`;
      if (level === 'error') {
        console.error(formatted, details || '');
      } else if (level === 'warn') {
        console.warn(formatted, details || '');
      } else {
        console.log(formatted, details || '');
      }
    }
  );

  // App Info handler
  ipcMain.handle(IPC_CHANNELS.APP_GET_INFO, async (): Promise<AppInfo> => {
    return {
      name: app.getName(),
      version: app.getVersion(),
      platform: process.platform,
      arch: process.arch,
      isDev: process.env.NODE_ENV === 'development' || !app.isPackaged,
    };
  });

  // Forward daemon status changes to renderer
  daemonManager.on('status-changed', (daemonInfo) => {
    const mainWindow = getMainWindow();
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send(IPC_CHANNELS.DAEMON_STATUS_CHANGED, daemonInfo);
    }
  });
}
