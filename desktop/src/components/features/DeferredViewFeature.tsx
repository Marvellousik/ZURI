import React from 'react';
import { Card } from '../primitives/Card';
import { Text } from '../primitives/Text';
import { Tag } from '../primitives/Tag';

export interface DeferredViewFeatureProps {
  tabName: string;
}

export const DeferredViewFeature: React.FC<DeferredViewFeatureProps> = ({ tabName }) => {
  return (
    <Card
      style={{
        flex: 1,
        padding: '48px 32px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        textAlign: 'center',
        gap: '16px',
      }}
    >
      <Tag>Deferred Modular View</Tag>

      <Text variant="display" style={{ fontSize: '20px' }}>
        {tabName.toUpperCase()} VIEW (DEFERRED TO SUBSEQUENT STAGE)
      </Text>

      <Text variant="body" color="secondary" style={{ maxWidth: '520px', textAlign: 'center' }}>
        This tab container is intentionally deferred per Stage 20 scope boundaries. Navigation remains modular and ready for instant view mounting in future sessions.
      </Text>
    </Card>
  );
};
