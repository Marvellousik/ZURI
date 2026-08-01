import React from 'react';
import { useNavigation } from '../hooks/useNavigation';
import { NavigationTab } from '../shared/types';
import { LayoutDashboard, Database, Activity, Settings } from 'lucide-react';
import { Text } from './primitives/Text';
import { Icon } from './primitives/Icon';
import { Tag } from './primitives/Tag';

interface NavItem {
  id: NavigationTab;
  label: string;
  icon: React.ComponentType<any>;
  stageNote?: string;
}

export const Navigation: React.FC = () => {
  const { activeTab, navigateTo } = useNavigation();

  const navItems: NavItem[] = [
    {
      id: 'dashboard',
      label: 'Daemon Overview',
      icon: LayoutDashboard,
    },
    {
      id: 'explorer',
      label: 'Memory Explorer',
      icon: Database,
      stageNote: 'Stage 20+',
    },
    {
      id: 'activity',
      label: 'Decision Log',
      icon: Activity,
      stageNote: 'Stage 20+',
    },
    {
      id: 'settings',
      label: 'System Settings',
      icon: Settings,
      stageNote: 'Stage 20+',
    },
  ];

  return (
    <nav
      style={{
        width: '240px',
        backgroundColor: 'var(--color-background)',
        borderRight: '1px solid var(--color-border)',
        display: 'flex',
        flexDirection: 'column',
        padding: '16px 12px',
        gap: '4px',
        flexShrink: 0,
      }}
    >
      <div
        style={{
          padding: '4px 12px 12px 12px',
        }}
      >
        <Text
          variant="caption"
          color="secondary"
          style={{ textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 600 }}
        >
          Navigation
        </Text>
      </div>

      {navItems.map((item) => {
        const isActive = activeTab === item.id;
        return (
          <button
            key={item.id}
            onClick={() => navigateTo(item.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              width: '100%',
              padding: '8px 12px',
              borderRadius: 'var(--radius-sm)',
              border: 'none',
              backgroundColor: isActive ? 'var(--color-primary-tint)' : 'transparent',
              color: isActive ? 'var(--color-primary)' : 'var(--color-text-secondary)',
              cursor: 'pointer',
              textAlign: 'left',
              transition: 'background-color var(--motion-fast) var(--easing-standard), color var(--motion-fast) var(--easing-standard)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <Icon
                icon={item.icon}
                size={18}
                style={{ color: isActive ? 'var(--color-primary)' : 'var(--color-muted)' }}
              />
              <Text
                variant={isActive ? 'body.medium' : 'body'}
                style={{ color: isActive ? 'var(--color-primary)' : 'var(--color-text-primary)' }}
              >
                {item.label}
              </Text>
            </div>
            {item.stageNote && <Tag>{item.stageNote}</Tag>}
          </button>
        );
      })}
    </nav>
  );
};
