import React from 'react';
import { Panel } from '../primitives/Panel';
import { Header } from '../Header';
import { DaemonControl } from '../DaemonControl';
import { DashboardView } from '../dashboard/DashboardView';
import { DeferredViewFeature } from '../features/DeferredViewFeature';
import { useNavigation } from '../../hooks/useNavigation';

export const MainPanel: React.FC = () => {
  const { activeTab } = useNavigation();

  const renderTabContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <DashboardView />;
      case 'explorer':
      case 'activity':
      case 'settings':
      default:
        return <DeferredViewFeature tabName={activeTab} />;
    }
  };

  return (
    <Panel variant="main">
      <Header />
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          padding: '24px',
          gap: '24px',
          overflowY: 'auto',
          animation: 'panelFadeIn var(--motion-fast) var(--easing-standard)',
        }}
      >
        <DaemonControl />
        {renderTabContent()}
      </div>
    </Panel>
  );
};
