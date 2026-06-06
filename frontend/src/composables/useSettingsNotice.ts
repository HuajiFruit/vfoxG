import { t } from '../i18n';

export type SettingsNotice = {
  type: 'success' | 'error';
  title: string;
  message: string;
};

export type SettingsNotifier = (notice: SettingsNotice) => void;

export const getErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err.trim()) return err;
  return fallback;
};

export const useSettingsNotice = (notify: SettingsNotifier) => {
  const notifyError = (message: string, title = t('common.error')) => {
    notify({ type: 'error', title, message });
  };

  const notifySuccess = (message: string, title = t('common.success')) => {
    notify({ type: 'success', title, message });
  };

  return {
    notifyError,
    notifySuccess,
  };
};
