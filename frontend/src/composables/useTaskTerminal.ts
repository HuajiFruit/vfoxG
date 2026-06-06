import { onMounted, onUnmounted, ref, type Ref } from 'vue';
import { t } from '../i18n';
import { onRuntimeEvent } from '../services/runtimeService';
import { isTaskErrorLog } from './taskLogParser';
import { formatDisplayError, formatTaskError, getErrorMessage } from './taskMessageFormat';
import { useTaskToast } from './useTaskToast';
import { useTerminalLogs } from './useTerminalLogs';

export type {
  NotifyPayload,
  TaskPhase,
  TerminalLogEntry,
  TerminalLogLevel,
  ToastStatus,
} from './taskTerminalTypes';

type UseTaskTerminalOptions = {
  showTerminal: Ref<boolean>;
  scrollTerminalToBottom: () => void;
};

export const useTaskTerminal = (options: UseTaskTerminalOptions) => {
  const terminalTaskRunning = ref(false);
  const taskToast = useTaskToast();
  const terminal = useTerminalLogs(options);

  let vfoxLogOff: (() => void) | null = null;
  let vfoxBusyOff: (() => void) | null = null;

  const runTerminalTask = async (title: string, task: () => Promise<void>) => {
    if (terminalTaskRunning.value) {
      taskToast.showTaskToast.value = true;
      taskToast.showBusyHint();
      return false;
    }

    terminalTaskRunning.value = true;
    taskToast.handleStartTask(title);
    try {
      await task();
      taskToast.completeRunningTaskSuccess();
      return true;
    } catch (err) {
      taskToast.completeRunningTaskError(formatDisplayError(getErrorMessage(err, t('toast.task_failed'))));
      throw err;
    } finally {
      terminalTaskRunning.value = false;
    }
  };

  const handleIdleErrorLog = (log: string) => {
    if (!isTaskErrorLog(log)) return;
    taskToast.handleNotify({
      type: 'error',
      title: t('common.error'),
      message: formatTaskError(log),
    });
  };

  const handleVfoxLog = (log: string) => {
    terminal.appendTerminalLog(log);

    if (!terminalTaskRunning.value && !taskToast.hasActiveTaskToast()) {
      handleIdleErrorLog(log);
      return;
    }

    taskToast.updateRunningTaskFromLog(log);
  };

  onMounted(() => {
    vfoxLogOff = onRuntimeEvent<string>('vfox-log', handleVfoxLog);
    vfoxBusyOff = onRuntimeEvent<void>('vfox-busy', () => {
      if (!terminalTaskRunning.value) return;
      taskToast.showTaskToast.value = true;
      taskToast.showBusyHint();
    });
  });

  onUnmounted(() => {
    if (vfoxLogOff) {
      vfoxLogOff();
      vfoxLogOff = null;
    }
    if (vfoxBusyOff) {
      vfoxBusyOff();
      vfoxBusyOff = null;
    }
    taskToast.disposeTaskToastTimers();
  });

  return {
    ...taskToast,
    ...terminal,
    terminalTaskRunning,
    runTerminalTask,
  };
};
