import { ref } from 'vue';
import { t } from '../i18n';
import {
  extractDownloadSpeed,
  extractProgressPercent,
  getInitialTaskPhase,
  getTaskPhaseFromLog,
  isDownloadThenInstallTask,
  isTaskDoneLog,
  isTaskErrorLog,
  isVersionNotReleasedMessage,
} from './taskLogParser';
import { formatDisplayError, formatTaskError } from './taskMessageFormat';
import type { NotifyPayload, TaskPhase, ToastStatus } from './taskTerminalTypes';
import { useTaskToastProgress } from './useTaskToastProgress';
import { useTaskToastTimers } from './useTaskToastTimers';

export const useTaskToast = () => {
  const showTaskToast = ref(false);
  const taskTitle = ref('');
  const taskStatus = ref<ToastStatus>('running');
  const lastLogLine = ref('');
  const taskHadError = ref(false);
  const taskPhase = ref<TaskPhase>('default');
  const taskUsesDownloadThenInstall = ref(false);
  const {
    busyHintVisible,
    clearAutoCloseTimer,
    scheduleToastClose,
    showBusyHint,
    disposeTaskToastTimers,
  } = useTaskToastTimers(showTaskToast);
  const {
    taskProgress,
    hasTaskProgress,
    taskDownloadSpeed,
    isDeterminateDownloadProgress,
    showToastProgress,
    toastProgressStyle,
    clearTaskProgress,
    setDownloadProgress,
    setCompleteProgress,
  } = useTaskToastProgress({ taskStatus, taskPhase });

  const hasActiveTaskToast = () => (
    showTaskToast.value &&
    taskStatus.value === 'running' &&
    taskTitle.value.trim() !== ''
  );

  const enterInstallPhase = () => {
    taskPhase.value = 'install';
    clearTaskProgress();
  };

  const handleStartTask = (title: string) => {
    taskTitle.value = title;
    lastLogLine.value = t('toast.starting');
    taskStatus.value = 'running';
    clearTaskProgress();
    taskHadError.value = false;
    taskUsesDownloadThenInstall.value = isDownloadThenInstallTask(title);
    taskPhase.value = getInitialTaskPhase(title);
    showTaskToast.value = true;
    clearAutoCloseTimer();
  };

  const completeRunningTaskSuccess = () => {
    if (taskStatus.value !== 'running') return;
    taskStatus.value = 'success';
    lastLogLine.value = t('toast.completed');
    setCompleteProgress();
    scheduleToastClose(2500);
  };

  const completeRunningTaskError = (message: string) => {
    if (taskStatus.value !== 'running') return;
    taskHadError.value = true;
    taskStatus.value = 'error';
    lastLogLine.value = formatDisplayError(message || t('toast.task_failed'));
    scheduleToastClose(5000);
  };

  const handleNotify = (payload: NotifyPayload) => {
    const notification = typeof payload === 'string'
      ? { message: payload, type: 'info' as const }
      : { type: 'info' as const, ...payload };

    taskTitle.value = notification.title || (
      notification.type === 'error'
        ? t('common.error')
        : notification.type === 'success'
          ? t('common.success')
          : t('common.notification')
    );
    lastLogLine.value = notification.type === 'error'
      ? formatDisplayError(notification.message)
      : notification.message;
    taskStatus.value = notification.type;
    if (notification.type === 'success') {
      setCompleteProgress();
    } else {
      clearTaskProgress();
    }
    taskHadError.value = notification.type === 'error';
    taskUsesDownloadThenInstall.value = false;
    taskPhase.value = 'default';
    showTaskToast.value = true;
    scheduleToastClose(notification.durationMs ?? (notification.type === 'error' ? 5000 : 3200));
  };

  const updateRunningTaskFromLog = (log: string) => {
    const parsedProgress = extractProgressPercent(log);
    const parsedSpeed = extractDownloadSpeed(log);
    const nextTaskPhase = getTaskPhaseFromLog(log);
    const shouldEnterInstallAfterDownload = taskUsesDownloadThenInstall.value && parsedProgress !== null && parsedProgress >= 100;
    let runningLogLine = log;

    if (taskStatus.value === 'running' && parsedProgress !== null && nextTaskPhase !== 'install' && !shouldEnterInstallAfterDownload) {
      taskPhase.value = 'download';
      setDownloadProgress(parsedProgress, parsedSpeed);
    }
    if (taskStatus.value === 'running' && (nextTaskPhase === 'install' || shouldEnterInstallAfterDownload)) {
      enterInstallPhase();
      runningLogLine = t('toast.installing_after_download');
    } else if (nextTaskPhase) {
      taskPhase.value = nextTaskPhase;
    }

    showTaskToast.value = true;

    if (isVersionNotReleasedMessage(log)) {
      taskHadError.value = true;
      taskStatus.value = 'error';
      lastLogLine.value = t('sdk.version_not_released');
    } else if (isTaskDoneLog(log)) {
      if (taskHadError.value) {
        scheduleToastClose(5000);
        return;
      }
      taskStatus.value = 'success';
      lastLogLine.value = t('toast.completed');
      setCompleteProgress();
    } else if (isTaskErrorLog(log)) {
      taskHadError.value = true;
      taskStatus.value = 'error';
      lastLogLine.value = formatTaskError(log);
    } else if (!taskHadError.value) {
      lastLogLine.value = runningLogLine;
    }

    if (taskStatus.value !== 'running') {
      scheduleToastClose(taskStatus.value === 'error' ? 5000 : 2500);
    }
  };

  return {
    showTaskToast,
    busyHintVisible,
    taskTitle,
    taskStatus,
    lastLogLine,
    taskProgress,
    taskDownloadSpeed,
    taskPhase,
    isDeterminateDownloadProgress,
    showToastProgress,
    toastProgressStyle,
    hasActiveTaskToast,
    showBusyHint,
    handleStartTask,
    handleNotify,
    completeRunningTaskSuccess,
    completeRunningTaskError,
    updateRunningTaskFromLog,
    disposeTaskToastTimers,
  };
};
