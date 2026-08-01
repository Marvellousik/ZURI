import { DaemonCommandResult, DaemonInfo } from '../shared/types';
import { daemonService, IDaemonService } from '../services/daemon-service';

export interface IDaemonRepository {
  getDaemonInfo(): Promise<DaemonInfo>;
  startDaemon(): Promise<DaemonCommandResult>;
  stopDaemon(): Promise<DaemonCommandResult>;
  restartDaemon(): Promise<DaemonCommandResult>;
  subscribeToStatus(callback: (info: DaemonInfo) => void): () => void;
}

export class DaemonRepository implements IDaemonRepository {
  constructor(private transportService: IDaemonService = daemonService) {}

  public async getDaemonInfo(): Promise<DaemonInfo> {
    return this.transportService.getStatus();
  }

  public async startDaemon(): Promise<DaemonCommandResult> {
    return this.transportService.start();
  }

  public async stopDaemon(): Promise<DaemonCommandResult> {
    return this.transportService.stop();
  }

  public async restartDaemon(): Promise<DaemonCommandResult> {
    return this.transportService.restart();
  }

  public subscribeToStatus(callback: (info: DaemonInfo) => void): () => void {
    return this.transportService.onStatusChanged(callback);
  }
}

export const daemonRepository = new DaemonRepository();
