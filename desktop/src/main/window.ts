import { BrowserWindow, app } from 'electron';
import path from 'path';

export class WindowManager {
  private mainWindow: BrowserWindow | null = null;

  public createMainWindow(): BrowserWindow {
    const isDev = process.env.NODE_ENV === 'development' || !app.isPackaged;

    this.mainWindow = new BrowserWindow({
      width: 1280,
      height: 830,
      minWidth: 1024,
      minHeight: 700,
      title: 'Zuri — Engineering Memory System',
      backgroundColor: '#090d16',
      show: false,
      webPreferences: {
        preload: path.join(__dirname, '../preload/index.js'),
        contextIsolation: true,
        nodeIntegration: false,
        sandbox: true,
        webSecurity: true,
      },
    });

    // Show when ready to avoid visual flickering
    this.mainWindow.once('ready-to-show', () => {
      this.mainWindow?.show();
    });

    if (isDev && process.env.VITE_DEV_SERVER_URL) {
      console.log(`[WindowManager] Loading URL: ${process.env.VITE_DEV_SERVER_URL}`);
      this.mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL);
      this.mainWindow.webContents.openDevTools({ mode: 'detach' });
    } else {
      const indexPath = path.join(__dirname, '../../dist/index.html');
      console.log(`[WindowManager] Loading File: ${indexPath}`);
      this.mainWindow.loadFile(indexPath);
    }

    this.mainWindow.on('closed', () => {
      this.mainWindow = null;
    });

    return this.mainWindow;
  }

  public getMainWindow(): BrowserWindow | null {
    return this.mainWindow;
  }
}
