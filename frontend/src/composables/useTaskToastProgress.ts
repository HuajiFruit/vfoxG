import { computed, ref, type Ref } from 'vue';
import type { TaskPhase, ToastStatus } from './taskTerminalTypes';

type UseTaskToastProgressOptions = {
  taskStatus: Ref<ToastStatus>;
  taskPhase: Ref<TaskPhase>;
};

export const useTaskToastProgress = ({ taskStatus, taskPhase }: UseTaskToastProgressOptions) => {
  const taskProgress = ref(0);
  const hasTaskProgress = ref(false);
  const taskDownloadSpeed = ref('');

  const clearTaskProgress = () => {
    taskProgress.value = 0;
    hasTaskProgress.value = false;
    taskDownloadSpeed.value = '';
  };

  const setDownloadProgress = (percent: number, downloadSpeed: string) => {
    taskProgress.value = percent;
    hasTaskProgress.value = true;
    if (downloadSpeed) {
      taskDownloadSpeed.value = downloadSpeed;
    }
  };

  const setCompleteProgress = () => {
    taskProgress.value = 100;
    hasTaskProgress.value = true;
    taskDownloadSpeed.value = '';
  };

  const isDeterminateDownloadProgress = computed(() => (
    taskStatus.value === 'running' &&
    taskPhase.value === 'download' &&
    hasTaskProgress.value
  ));

  const showToastProgress = computed(() => (
    taskStatus.value === 'running' ||
    taskStatus.value === 'success' ||
    (taskStatus.value === 'error' && hasTaskProgress.value)
  ));

  const toastProgressStyle = computed(() => {
    if (isDeterminateDownloadProgress.value) return { width: `${taskProgress.value}%` };
    if (taskStatus.value === 'success') return { width: '100%' };
    if (taskStatus.value === 'error' && hasTaskProgress.value) return { width: `${taskProgress.value}%` };
    return undefined;
  });

  return {
    taskProgress,
    hasTaskProgress,
    taskDownloadSpeed,
    isDeterminateDownloadProgress,
    showToastProgress,
    toastProgressStyle,
    clearTaskProgress,
    setDownloadProgress,
    setCompleteProgress,
  };
};
