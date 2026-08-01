import { app, BrowserWindow } from 'electron';
import { WindowManager } from './window';
import { DaemonProcessManager } from './daemon';
import { registerIPCHandlers } from '../ipc/main-handlers';
import { setupApplicationMenu } from './menu';

let windowManager: WindowManager | null = null;
let daemonManager: DaemonProcessManager | null = null;

async function bootstrap() {
  console.log('[Main Process] Initializing Zuri Desktop Application Shell...');

  windowManager = new WindowManager();
  daemonManager = new DaemonProcessManager();

  // Setup application menu
  setupApplicationMenu(daemonManager);

  // Register IPC handlers
  registerIPCHandlers(daemonManager, () => windowManager?.getMainWindow() || null);

  // Create main window
  const mainWindow = windowManager.createMainWindow();

  // Start background monitoring of daemon status
  daemonManager.startMonitoring(3000);

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0 && windowManager) {
      windowManager.createMainWindow();
    }
  });
}

app.whenReady().then(bootstrap);

app.on('window-all-closed', async () => {
  console.log('[Main Process] All windows closed.');
  if (daemonManager) {
    daemonManager.stopMonitoring();
    // Stop daemon on app quit
    await daemonManager.stop();
  }
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', async () => {
  if (daemonManager) {
    daemonManager.stopMonitoring();
    await daemonManager.stop();
  }
});
