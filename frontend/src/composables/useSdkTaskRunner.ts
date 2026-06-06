import { t } from '../i18n';

export type RunTerminalTask = (title: string, task: () => Promise<void>) => Promise<boolean>;

type UseSdkTaskRunnerOptions = {
  runTerminalTask?: RunTerminalTask;
  emitStartTask: (title: string) => void;
  notifyInfo: (message: string) => void;
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

const isBusyErrorMessage = (message: string) =>
  message.toLowerCase().includes('another terminal task is already running');

export const useSdkTaskRunner = (options: UseSdkTaskRunnerOptions) => {
  const runTask = async (title: string, task: () => Promise<void>) => {
    if (options.runTerminalTask) {
      return options.runTerminalTask(title, task);
    }
    options.emitStartTask(title);
    await task();
    return true;
  };

  const notifyTaskError = (err: unknown, fallback: string) => {
    const message = options.formatError(err, fallback);
    if (isBusyErrorMessage(options.formatError(err, ''))) {
      options.notifyInfo(t('toast.please_wait') || fallback);
      return;
    }
    options.notifyError(message);
  };

  return {
    runTask,
    notifyTaskError,
  };
};
