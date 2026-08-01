import React from 'react';
import ReactDOM from 'react-dom/client';
import { AppProvider, useAppState } from '../state/AppContext';
import { AppLayout } from '../layouts/AppLayout';
import { LoadingScreen } from '../components/LoadingScreen';
import './index.css';

const AppShell: React.FC = () => {
  const { isInitialLoading, theme } = useAppState();

  if (isInitialLoading) {
    return <LoadingScreen />;
  }

  return (
    <div data-theme={theme}>
      <AppLayout />
    </div>
  );
};

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <AppProvider>
      <AppShell />
    </AppProvider>
  </React.StrictMode>
);
