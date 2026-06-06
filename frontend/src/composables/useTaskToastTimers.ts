import { ref, type Ref } from 'vue';

export const useTaskToastTimers = (showTaskToast: Ref<boolean>) => {
  const busyHintVisible = ref(false);
  let autoCloseTimer: ReturnType<typeof setTimeout> | null = null;
  let busyHintTimer: ReturnType<typeof setTimeout> | null = null;

  const clearAutoCloseTimer = () => {
    if (!autoCloseTimer) return;
    clearTimeout(autoCloseTimer);
    autoCloseTimer = null;
  };

  const scheduleToastClose = (durationMs: number) => {
    clearAutoCloseTimer();
    autoCloseTimer = setTimeout(() => {
      showTaskToast.value = false;
      autoCloseTimer = null;
    }, durationMs);
  };

  const showBusyHint = () => {
    busyHintVisible.value = true;
    if (busyHintTimer) clearTimeout(busyHintTimer);
    busyHintTimer = setTimeout(() => {
      busyHintVisible.value = false;
      busyHintTimer = null;
    }, 1800);
  };

  const disposeTaskToastTimers = () => {
    clearAutoCloseTimer();
    if (busyHintTimer) {
      clearTimeout(busyHintTimer);
      busyHintTimer = null;
    }
  };

  return {
    busyHintVisible,
    clearAutoCloseTimer,
    scheduleToastClose,
    showBusyHint,
    disposeTaskToastTimers,
  };
};
