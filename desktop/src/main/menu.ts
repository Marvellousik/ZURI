import { Menu, MenuItemConstructorOptions, app, shell } from 'electron';
import { DaemonProcessManager } from './daemon';

export function setupApplicationMenu(daemonManager: DaemonProcessManager): void {
  const isMac = process.platform === 'darwin';

  const template: MenuItemConstructorOptions[] = [
    ...(isMac
      ? [
          {
            label: app.name,
            submenu: [
              { role: 'about' as const },
              { type: 'separator' as const },
              { role: 'services' as const },
              { type: 'separator' as const },
              { role: 'hide' as const },
              { role: 'hideOthers' as const },
              { role: 'unhide' as const },
              { type: 'separator' as const },
              { role: 'quit' as const },
            ],
          },
        ]
      : []),
    {
      label: 'File',
      submenu: [isMac ? { role: 'close' as const } : { role: 'quit' as const }],
    },
    {
      label: 'Daemon',
      submenu: [
        {
          label: 'Start Daemon',
          accelerator: 'CmdOrCtrl+Shift+S',
          click: () => daemonManager.start(),
        },
        {
          label: 'Stop Daemon',
          accelerator: 'CmdOrCtrl+Shift+X',
          click: () => daemonManager.stop(),
        },
        {
          label: 'Restart Daemon',
          accelerator: 'CmdOrCtrl+Shift+R',
          click: () => daemonManager.restart(),
        },
        { type: 'separator' as const },
        {
          label: 'Check Daemon Health',
          click: () => daemonManager.checkHealth(),
        },
      ],
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' as const },
        { role: 'forceReload' as const },
        { role: 'toggleDevTools' as const },
        { type: 'separator' as const },
        { role: 'resetZoom' as const },
        { role: 'zoomIn' as const },
        { role: 'zoomOut' as const },
        { type: 'separator' as const },
        { role: 'togglefullscreen' as const },
      ],
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'Zuri Spec & Documentation',
          click: async () => {
            await shell.openExternal('https://github.com/Marvellousik/ZURI');
          },
        },
      ],
    },
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}
