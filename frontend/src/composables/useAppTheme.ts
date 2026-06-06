import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue';
import { setWindowThemeMode } from '../services/runtimeService';

type AppTheme = 'auto' | 'light' | 'dark';

type UseAppThemeOptions = {
  onShowTerminal?: () => void;
};

const themeStorageKey = 'vfox-theme';
const terminalStorageKey = 'vfox-show-terminal';

const readThemeSetting = (): AppTheme => {
  const storedTheme = localStorage.getItem(themeStorageKey);
  return storedTheme === 'light' || storedTheme === 'dark' || storedTheme === 'auto'
    ? storedTheme
    : 'auto';
};

export const useAppTheme = (options: UseAppThemeOptions = {}) => {
  const theme = ref<AppTheme>(readThemeSetting());
  const showTerminal = ref(localStorage.getItem(terminalStorageKey) === 'true');
  const systemThemeQuery = window.matchMedia('(prefers-color-scheme: dark)');

  const applyTheme = () => {
    if (theme.value === 'auto') {
      const isDarkTheme = systemThemeQuery.matches;
      document.documentElement.setAttribute('data-theme', isDarkTheme ? 'dark' : 'light');
      setWindowThemeMode('auto');
      return;
    }

    document.documentElement.setAttribute('data-theme', theme.value);
    setWindowThemeMode(theme.value);
  };

  const handleSystemThemeChange = () => {
    if (theme.value === 'auto') {
      applyTheme();
    }
  };

  watch(theme, (newTheme) => {
    localStorage.setItem(themeStorageKey, newTheme);
    applyTheme();
  });

  watch(showTerminal, (isTerminalVisible) => {
    localStorage.setItem(terminalStorageKey, String(isTerminalVisible));
    if (isTerminalVisible) {
      options.onShowTerminal?.();
    }
  });

  onMounted(() => {
    applyTheme();
    systemThemeQuery.addEventListener('change', handleSystemThemeChange);
  });

  onUnmounted(() => {
    systemThemeQuery.removeEventListener('change', handleSystemThemeChange);
  });

  return {
    theme: theme as Ref<string>,
    showTerminal,
  };
};
