export type ThemeMode = 'dark' | 'light';

export interface ISettingsRepository {
  getTheme(): ThemeMode;
  setTheme(theme: ThemeMode): void;
}

export class SettingsRepository implements ISettingsRepository {
  private readonly THEME_KEY = 'zuri_app_theme';

  public getTheme(): ThemeMode {
    if (typeof window !== 'undefined' && window.localStorage) {
      const saved = window.localStorage.getItem(this.THEME_KEY);
      if (saved === 'light' || saved === 'dark') {
        return saved;
      }
    }
    return 'dark';
  }

  public setTheme(theme: ThemeMode): void {
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem(this.THEME_KEY, theme);
    }
  }
}

export const settingsRepository = new SettingsRepository();
