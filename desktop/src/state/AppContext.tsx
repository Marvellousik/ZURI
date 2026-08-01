import React, { createContext, useContext, useEffect, useState } from 'react';
import { DaemonInfo, NavigationTab } from '../shared/types';
import { daemonRepository, IDaemonRepository } from '../repositories/daemon-repository';
import { settingsRepository, ISettingsRepository, ThemeMode } from '../repositories/settings-repository';
import { errorHandler } from '../services/error-service';
import { logger } from '../services/logger-service';

export interface AppState {
  daemonInfo: DaemonInfo;
  activeTab: NavigationTab;
  theme: ThemeMode;
  isInitialLoading: boolean;
  lastError: string | null;
  setActiveTab: (tab: NavigationTab) => void;
  setTheme: (theme: ThemeMode) => void;
  startDaemon: () => Promise<void>;
  stopDaemon: () => Promise<void>;
  restartDaemon: () => Promise<void>;
  refreshDaemonStatus: () => Promise<void>;
  clearError: () => void;
}

const initialDaemonInfo: DaemonInfo = {
  status: 'STOPPED',
  port: 7331,
  host: '127.0.0.1',
  lastCheckedAt: new Date().toISOString(),
};

interface AppProviderProps {
  children: React.ReactNode;
  daemonRepo?: IDaemonRepository;
  settingsRepo?: ISettingsRepository;
}

const AppContext = createContext<AppState | undefined>(undefined);

export const AppProvider: React.FC<AppProviderProps> = ({
  children,
  daemonRepo = daemonRepository,
  settingsRepo = settingsRepository,
}) => {
  const [daemonInfo, setDaemonInfo] = useState<DaemonInfo>(initialDaemonInfo);
  const [activeTab, setActiveTab] = useState<NavigationTab>('dashboard');
  const [theme, setThemeState] = useState<ThemeMode>(() => settingsRepo.getTheme());
  const [isInitialLoading, setIsInitialLoading] = useState<boolean>(true);
  const [lastError, setLastError] = useState<string | null>(null);

  useEffect(() => {
    // Global error listener
    errorHandler.initGlobalHandlers();
    const unsubscribeError = errorHandler.subscribe((err) => {
      setLastError(`[${err.source}] ${err.message}`);
    });

    // Initial daemon status fetch from repository
    const initStatus = async () => {
      try {
        const info = await daemonRepo.getDaemonInfo();
        setDaemonInfo(info);
        logger.info(`Initialized daemon status via DaemonRepository: ${info.status}`);
      } catch (err: any) {
        errorHandler.handleError(err, 'AppProvider.initStatus');
      } finally {
        setIsInitialLoading(false);
      }
    };

    initStatus();

    // Subscribe to IPC daemon status pushes via Repository
    const unsubscribeStatus = daemonRepo.subscribeToStatus((newInfo) => {
      setDaemonInfo(newInfo);
    });

    return () => {
      unsubscribeError();
      unsubscribeStatus();
    };
  }, [daemonRepo]);

  const handleSetTheme = (newTheme: ThemeMode) => {
    settingsRepo.setTheme(newTheme);
    setThemeState(newTheme);
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', newTheme);
    }
  };

  useEffect(() => {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', theme);
    }
  }, [theme]);

  const handleStartDaemon = async () => {
    setLastError(null);
    const result = await daemonRepo.startDaemon();
    if (!result.success && result.error) {
      setLastError(result.error);
    }
    if (result.daemonInfo) {
      setDaemonInfo(result.daemonInfo);
    }
  };

  const handleStopDaemon = async () => {
    setLastError(null);
    const result = await daemonRepo.stopDaemon();
    if (!result.success && result.error) {
      setLastError(result.error);
    }
    if (result.daemonInfo) {
      setDaemonInfo(result.daemonInfo);
    }
  };

  const handleRestartDaemon = async () => {
    setLastError(null);
    const result = await daemonRepo.restartDaemon();
    if (!result.success && result.error) {
      setLastError(result.error);
    }
    if (result.daemonInfo) {
      setDaemonInfo(result.daemonInfo);
    }
  };

  const handleRefreshDaemonStatus = async () => {
    const info = await daemonRepo.getDaemonInfo();
    setDaemonInfo(info);
  };

  const clearError = () => {
    setLastError(null);
  };

  return (
    <AppContext.Provider
      value={{
        daemonInfo,
        activeTab,
        theme,
        isInitialLoading,
        lastError,
        setActiveTab,
        setTheme: handleSetTheme,
        startDaemon: handleStartDaemon,
        stopDaemon: handleStopDaemon,
        restartDaemon: handleRestartDaemon,
        refreshDaemonStatus: handleRefreshDaemonStatus,
        clearError,
      }}
    >
      {children}
    </AppContext.Provider>
  );
};

export const useAppState = (): AppState => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useAppState must be used within an AppProvider');
  }
  return context;
};
