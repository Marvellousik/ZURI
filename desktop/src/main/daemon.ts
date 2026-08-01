import { exec, spawn, ChildProcess } from 'child_process';
import http from 'http';
import path from 'path';
import fs from 'fs';
import EventEmitter from 'events';
import { DaemonCommandResult, DaemonInfo, DaemonStatus } from '../shared/types';

export class DaemonProcessManager extends EventEmitter {
  private status: DaemonStatus = 'STOPPED';
  private process: ChildProcess | null = null;
  private port: number = 7331;
  private host: string = '127.0.0.1';
  private healthCheckInterval: NodeJS.Timeout | null = null;
  private startTime: number | null = null;
  private lastError: string | undefined = undefined;

  constructor() {
    super();
    const envPort = process.env.ZURI_PORT;
    if (envPort) {
      const parsed = parseInt(envPort, 10);
      if (!isNaN(parsed)) this.port = parsed;
    }
    const envHost = process.env.ZURI_HOST;
    if (envHost) this.host = envHost;
  }

  public getDaemonInfo(): DaemonInfo {
    const uptimeSeconds = this.startTime && this.status === 'RUNNING' 
      ? Math.floor((Date.now() - this.startTime) / 1000)
      : undefined;

    return {
      status: this.status,
      port: this.port,
      host: this.host,
      pid: this.process?.pid,
      uptimeSeconds,
      error: this.lastError,
      lastCheckedAt: new Date().toISOString(),
    };
  }

  public startMonitoring(intervalMs: number = 3000): void {
    if (this.healthCheckInterval) clearInterval(this.healthCheckInterval);
    
    // Initial check
    this.checkHealth();
    
    this.healthCheckInterval = setInterval(() => {
      this.checkHealth();
    }, intervalMs);
  }

  public stopMonitoring(): void {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
      this.healthCheckInterval = null;
    }
  }

  public async checkHealth(): Promise<boolean> {
    return new Promise((resolve) => {
      const req = http.get(
        `http://${this.host}:${this.port}/health`,
        { timeout: 2000 },
        (res) => {
          if (res.statusCode === 200) {
            if (this.status !== 'RUNNING') {
              this.updateStatus('RUNNING');
              if (!this.startTime) this.startTime = Date.now();
            }
            resolve(true);
          } else {
            if (this.status === 'RUNNING') {
              this.updateStatus('ERROR', `Health check returned status code ${res.statusCode}`);
            }
            resolve(false);
          }
        }
      );

      req.on('error', (err) => {
        if (this.status === 'RUNNING') {
          this.updateStatus('STOPPED', err.message);
        } else if (this.status === 'STARTING') {
          // Keep STARTING until timeout or process exit
        }
        resolve(false);
      });

      req.end();
    });
  }

  public async start(): Promise<DaemonCommandResult> {
    if (this.status === 'RUNNING' || this.status === 'STARTING') {
      return {
        success: false,
        error: `Daemon is already ${this.status.toLowerCase()}`,
        daemonInfo: this.getDaemonInfo(),
      };
    }

    this.updateStatus('STARTING');
    this.lastError = undefined;

    try {
      const isDev = process.env.NODE_ENV === 'development' || !process.env.NODE_ENV;
      const cwd = process.cwd();
      
      // Resolve Go repository root (handles execution from desktop/ workspace or repo root)
      const repoRoot = fs.existsSync(path.join(cwd, 'go.mod'))
        ? cwd
        : path.resolve(cwd, '..');

      const exeName = process.platform === 'win32' ? 'zuri-daemon.exe' : 'zuri-daemon';
      const daemonExePath = path.join(repoRoot, exeName);

      if (fs.existsSync(daemonExePath)) {
        console.log(`[DaemonProcessManager] Spawning daemon executable from: ${daemonExePath}`);
        this.process = spawn(daemonExePath, [], {
          cwd: repoRoot,
          env: {
            ...process.env,
            ZURI_PORT: String(this.port),
            ZURI_HOST: this.host,
          },
          stdio: ['ignore', 'pipe', 'pipe'],
        });
      } else if (isDev) {
        console.log(`[DaemonProcessManager] Executable not found at ${daemonExePath}. Spawning 'go run ./cmd/daemon' in ${repoRoot}...`);
        this.process = spawn('go', ['run', './cmd/daemon'], {
          cwd: repoRoot,
          env: {
            ...process.env,
            ZURI_PORT: String(this.port),
            ZURI_HOST: this.host,
          },
          stdio: ['ignore', 'pipe', 'pipe'],
        });
      } else {
        throw new Error(`Daemon executable not found at: ${daemonExePath}`);
      }

      if (this.process) {
        this.process.stdout?.on('data', (data) => {
          console.log(`[Daemon STDOUT] ${data.toString().trim()}`);
        });

        this.process.stderr?.on('data', (data) => {
          console.error(`[Daemon STDERR] ${data.toString().trim()}`);
        });

        this.process.on('exit', (code, signal) => {
          console.log(`[DaemonProcessManager] Process exited with code ${code}, signal ${signal}`);
          this.process = null;
          this.startTime = null;
          if (this.status !== 'STOPPING') {
            this.updateStatus('STOPPED', code !== 0 ? `Exited with code ${code}` : undefined);
          } else {
            this.updateStatus('STOPPED');
          }
        });

        this.process.on('error', (err) => {
          console.error(`[DaemonProcessManager] Failed to start process:`, err);
          this.process = null;
          this.startTime = null;
          this.updateStatus('ERROR', err.message);
        });
      }

      // Poll health endpoint for up to 10 seconds to confirm startup
      const healthy = await this.pollUntilHealthy(10000);
      if (healthy) {
        this.updateStatus('RUNNING');
        this.startTime = Date.now();
        return {
          success: true,
          message: 'Daemon started successfully',
          daemonInfo: this.getDaemonInfo(),
        };
      } else {
        return {
          success: false,
          error: 'Daemon process started but health check timed out',
          daemonInfo: this.getDaemonInfo(),
        };
      }
    } catch (err: any) {
      this.updateStatus('ERROR', err.message);
      return {
        success: false,
        error: err.message,
        daemonInfo: this.getDaemonInfo(),
      };
    }
  }

  public async stop(): Promise<DaemonCommandResult> {
    if (this.status === 'STOPPED') {
      return {
        success: true,
        message: 'Daemon is already stopped',
        daemonInfo: this.getDaemonInfo(),
      };
    }

    this.updateStatus('STOPPING');

    if (this.process) {
      return new Promise((resolve) => {
        const proc = this.process!;
        const timeout = setTimeout(() => {
          if (proc && !proc.killed) {
            console.warn('[DaemonProcessManager] Graceful stop timed out. Killing process...');
            proc.kill('SIGKILL');
          }
        }, 5000);

        proc.once('exit', () => {
          clearTimeout(timeout);
          this.process = null;
          this.startTime = null;
          this.updateStatus('STOPPED');
          resolve({
            success: true,
            message: 'Daemon stopped successfully',
            daemonInfo: this.getDaemonInfo(),
          });
        });

        proc.kill('SIGTERM');
      });
    } else {
      this.updateStatus('STOPPED');
      return {
        success: true,
        message: 'Daemon process cleared',
        daemonInfo: this.getDaemonInfo(),
      };
    }
  }

  public async restart(): Promise<DaemonCommandResult> {
    console.log('[DaemonProcessManager] Restarting daemon...');
    await this.stop();
    await new Promise((r) => setTimeout(r, 1000));
    return this.start();
  }

  private updateStatus(newStatus: DaemonStatus, error?: string): void {
    this.status = newStatus;
    if (error !== undefined) this.lastError = error;
    this.emit('status-changed', this.getDaemonInfo());
  }

  private async pollUntilHealthy(maxTimeoutMs: number): Promise<boolean> {
    const startTime = Date.now();
    while (Date.now() - startTime < maxTimeoutMs) {
      const isHealthy = await this.checkHealth();
      if (isHealthy) return true;
      await new Promise((r) => setTimeout(r, 500));
    }
    return false;
  }
}
