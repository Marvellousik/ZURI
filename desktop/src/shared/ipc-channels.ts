/**
 * Strongly typed IPC Channel constants shared between Main, Preload, and Renderer processes.
 */
export const IPC_CHANNELS = {
  // Daemon lifecycle channels
  DAEMON_GET_STATUS: 'daemon:get-status',
  DAEMON_START: 'daemon:start',
  DAEMON_STOP: 'daemon:stop',
  DAEMON_RESTART: 'daemon:restart',
  DAEMON_STATUS_CHANGED: 'daemon:status-changed',

  // Logging and telemetry channels
  LOG_MESSAGE: 'log:message',

  // System/App controls
  APP_GET_INFO: 'app:get-info',
} as const;

export type IPCChannelName = typeof IPC_CHANNELS[keyof typeof IPC_CHANNELS];
