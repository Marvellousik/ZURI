import { useAppState } from '../state/AppContext';

export function useTheme() {
  const { theme, setTheme } = useAppState();

  const toggleTheme = () => {
    setTheme(theme === 'dark' ? 'light' : 'dark');
  };

  return {
    theme,
    isDark: theme === 'dark',
    setTheme,
    toggleTheme,
  };
}
