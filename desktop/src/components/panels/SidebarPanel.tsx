import React from 'react';
import { Panel } from '../primitives/Panel';
import { Navigation } from '../Navigation';

export const SidebarPanel: React.FC = () => {
  return (
    <Panel variant="sidebar">
      <Navigation />
    </Panel>
  );
};
