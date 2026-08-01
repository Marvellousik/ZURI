import { useAppState } from '../state/AppContext';
import { NavigationTab } from '../shared/types';

export function useNavigation() {
  const { activeTab, setActiveTab } = useAppState();

  return {
    activeTab,
    navigateTo: (tab: NavigationTab) => setActiveTab(tab),
    isCurrentTab: (tab: NavigationTab) => activeTab === tab,
  };
}
