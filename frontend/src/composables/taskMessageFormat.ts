import { t } from '../i18n';
import { getRawTaskErrorMessage, isVersionNotReleasedMessage } from './taskLogParser';

export const getErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err.trim()) return err;
  return fallback;
};

export const formatDisplayError = (message: string) => {
  const cleanMessage = message.trim();
  if (isVersionNotReleasedMessage(cleanMessage)) {
    return t('sdk.version_not_released');
  }
  return cleanMessage || t('toast.task_failed');
};

export const formatTaskError = (log: string) => formatDisplayError(getRawTaskErrorMessage(log));
