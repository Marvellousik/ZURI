import { useAppState } from '../state/AppContext';

export function useDaemonStatus() {
  const { daemonInfo, startDaemon, stopDaemon, restartDaemon, refreshDaemonStatus } = useAppState();

  return {
    status: daemonInfo.status,
    port: daemonInfo.port,
    host: daemonInfo.host,
    pid: daemonInfo.pid,
    uptimeSeconds: daemonInfo.uptimeSeconds,
    error: daemonInfo.error,
    lastCheckedAt: daemonInfo.lastCheckedAt,
    isRunning: daemonInfo.status === 'RUNNING',
    isStarting: daemonInfo.status === 'STARTING',
    isStopped: daemonInfo.status === 'STOPPED',
    isStopping: daemonInfo.status === 'STOPPING',
    isError: daemonInfo.status === 'ERROR',
    start: startDaemon,
    stop: stopDaemon,
    restart: restartDaemon,
    refresh: refreshDaemonStatus,
  };
}
