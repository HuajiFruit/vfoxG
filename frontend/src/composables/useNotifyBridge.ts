import { t } from '../i18n';

type NotifyType = 'error' | 'success' | 'info';

type NotifyBridgeOptions = {
  emitNotify: (payload: { type: NotifyType; title: string; message: string }) => void;
};

export const getErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err.trim()) return err;
  return fallback;
};

export const useNotifyBridge = ({ emitNotify }: NotifyBridgeOptions) => {
  const notifyError = (message: string, title = t('common.error')) => {
    emitNotify({ type: 'error', title, message });
  };

  const notifySuccess = (message: string, title = t('common.success')) => {
    emitNotify({ type: 'success', title, message });
  };

  const notifyInfo = (message: string, title = t('common.notification')) => {
    emitNotify({ type: 'info', title, message });
  };

  return {
    notifyError,
    notifySuccess,
    notifyInfo,
    getErrorMessage,
  };
};
