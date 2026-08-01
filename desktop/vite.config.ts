import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import electron from 'vite-plugin-electron';
import renderer from 'vite-plugin-electron-renderer';
import path from 'path';

export default defineConfig({
  plugins: [
    react(),
    electron([
      {
        entry: 'src/main/index.ts',
        vite: {
          build: {
            outDir: 'dist-electron/main',
            rollupOptions: {
              external: ['electron'],
            },
          },
          resolve: {
            alias: {
              '@shared': path.resolve(import.meta.dirname, 'src/shared'),
              '@main': path.resolve(import.meta.dirname, 'src/main'),
              '@ipc': path.resolve(import.meta.dirname, 'src/ipc'),
              '@services': path.resolve(import.meta.dirname, 'src/services'),
              '@repositories': path.resolve(import.meta.dirname, 'src/repositories'),
            },
          },
        },
      },
      {
        entry: 'src/preload/index.ts',
        onstart(options) {
          options.reload();
        },
        vite: {
          build: {
            outDir: 'dist-electron/preload',
            rollupOptions: {
              external: ['electron'],
            },
          },
          resolve: {
            alias: {
              '@shared': path.resolve(import.meta.dirname, 'src/shared'),
              '@preload': path.resolve(import.meta.dirname, 'src/preload'),
            },
          },
        },
      },
    ]),
    renderer(),
  ],
  resolve: {
    alias: {
      '@shared': path.resolve(import.meta.dirname, 'src/shared'),
      '@renderer': path.resolve(import.meta.dirname, 'src/renderer'),
      '@components': path.resolve(import.meta.dirname, 'src/components'),
      '@layouts': path.resolve(import.meta.dirname, 'src/layouts'),
      '@hooks': path.resolve(import.meta.dirname, 'src/hooks'),
      '@services': path.resolve(import.meta.dirname, 'src/services'),
      '@repositories': path.resolve(import.meta.dirname, 'src/repositories'),
      '@ipc': path.resolve(import.meta.dirname, 'src/ipc'),
      '@state': path.resolve(import.meta.dirname, 'src/state'),
      '@assets': path.resolve(import.meta.dirname, 'src/assets'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
