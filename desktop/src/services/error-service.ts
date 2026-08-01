import { logger } from './logger-service';

export interface AppError {
  id: string;
  message: string;
  timestamp: string;
  stack?: string;
  source: string;
}

class ErrorHandlerService {
  private errorListeners: Set<(error: AppError) => void> = new Set();

  public initGlobalHandlers(): void {
    if (typeof window === 'undefined') return;

    window.addEventListener('error', (event) => {
      this.handleError(
        event.error || new Error(event.message),
        `Unhandled Window Error (${event.filename}:${event.lineno})`
      );
    });

    window.addEventListener('unhandledrejection', (event) => {
      const reason = event.reason instanceof Error ? event.reason : new Error(String(event.reason));
      this.handleError(reason, 'Unhandled Promise Rejection');
    });
  }

  public handleError(error: Error | string, source: string = 'Application'): AppError {
    const message = typeof error === 'string' ? error : error.message;
    const stack = typeof error === 'string' ? undefined : error.stack;

    const appError: AppError = {
      id: Math.random().toString(36).substring(2, 9),
      message,
      timestamp: new Date().toISOString(),
      stack,
      source,
    };

    logger.error(`[${source}] ${message}`, { stack });

    this.errorListeners.forEach((listener) => {
      try {
        listener(appError);
      } catch (e) {
        console.error('Error in error listener:', e);
      }
    });

    return appError;
  }

  public subscribe(listener: (error: AppError) => void): () => void {
    this.errorListeners.add(listener);
    return () => this.errorListeners.delete(listener);
  }
}

export const errorHandler = new ErrorHandlerService();
