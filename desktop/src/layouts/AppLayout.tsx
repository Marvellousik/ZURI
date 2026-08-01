import React from 'react';
import { SidebarPanel } from '../components/panels/SidebarPanel';
import { MainPanel } from '../components/panels/MainPanel';
import { ActivityRailPanel } from '../components/panels/ActivityRailPanel';

export const AppLayout: React.FC = () => {
  return (
    <div
      style={{
        display: 'flex',
        height: '100vh',
        width: '100vw',
        overflow: 'hidden',
        backgroundColor: 'var(--color-background)',
        color: 'var(--color-text-primary)',
      }}
    >
      {/* 240px Fixed Sidebar Panel */}
      <SidebarPanel />

      {/* Flexible Main Content Panel */}
      <MainPanel />

      {/* 320px Right Rail Activity Feed Panel */}
      <ActivityRailPanel />
    </div>
  );
};
