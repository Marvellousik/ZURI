import { LogEntry, LogLevel } from '../shared/types';

class LoggerService {
  private logs: LogEntry[] = [];
  private maxLogs = 500;
  private listeners: Set<(log: LogEntry) => void> = new Set();

  public log(level: LogLevel, message: string, details?: unknown, source: 'renderer' | 'main' | 'daemon' = 'renderer'): LogEntry {
    const entry: LogEntry = {
      id: Math.random().toString(36).substring(2, 9),
      timestamp: new Date().toISOString(),
      level,
      message,
      source,
      details,
    };

    this.logs.unshift(entry);
    if (this.logs.length > this.maxLogs) {
      this.logs.pop();
    }

    // Pass log to main process logger if available
    if (typeof window !== 'undefined' && window.zuriAPI?.logMessage) {
      window.zuriAPI.logMessage(level, message, details).catch(() => {});
    }

    this.notifyListeners(entry);
    return entry;
  }

  public info(message: string, details?: unknown): LogEntry {
    return this.log('info', message, details);
  }

  public warn(message: string, details?: unknown): LogEntry {
    return this.log('warn', message, details);
  }

  public error(message: string, details?: unknown): LogEntry {
    return this.log('error', message, details);
  }

  public getLogs(): LogEntry[] {
    return [...this.logs];
  }

  public subscribe(listener: (log: LogEntry) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private notifyListeners(entry: LogEntry): void {
    this.listeners.forEach((listener) => {
      try {
        listener(entry);
      } catch (err) {
        console.error('Error in logger listener:', err);
      }
    });
  }
}

export const logger = new LoggerService();
