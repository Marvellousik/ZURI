import React from 'react';
import { Panel } from '../primitives/Panel';
import { ActivityFeedFeature } from '../features/ActivityFeedFeature';

export const ActivityRailPanel: React.FC = () => {
  return (
    <Panel variant="rail">
      <ActivityFeedFeature />
    </Panel>
  );
};
