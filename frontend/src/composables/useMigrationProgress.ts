import { onMounted, onUnmounted, ref } from 'vue';
import { t } from '../i18n';
import { onRuntimeEvent } from '../services/runtimeService';

export type MigrationProgress = {
  stage: 'preparing' | 'copying' | 'done' | 'error' | string;
  current: string;
  completed: number;
  total: number;
  percent: number;
  estimatedRemaining: number;
};

const clampProgressPercent = (percent: number) => Math.max(0, Math.min(100, percent || 0));

export const formatMigrationRemainingTime = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds <= 0) return t('migration.estimate.calculating');
  if (seconds < 60) return t('migration.estimate.seconds', { seconds });
  return t('migration.estimate.minutes', { minutes: Math.ceil(seconds / 60) });
};

export const useMigrationProgress = () => {
  const migrationVisible = ref(false);
  const migrationProgress = ref<MigrationProgress>({
    stage: 'preparing',
    current: '',
    completed: 0,
    total: 0,
    percent: 0,
    estimatedRemaining: 0,
  });

  let migrationProgressOff: (() => void) | null = null;
  let migrationCloseTimer: ReturnType<typeof setTimeout> | null = null;

  onMounted(() => {
    migrationProgressOff = onRuntimeEvent<MigrationProgress>('migration-progress', (progress) => {
      if (migrationCloseTimer) {
        clearTimeout(migrationCloseTimer);
        migrationCloseTimer = null;
      }

      migrationProgress.value = {
        stage: progress.stage || 'copying',
        current: progress.current || '',
        completed: progress.completed || 0,
        total: progress.total || 0,
        percent: clampProgressPercent(progress.percent),
        estimatedRemaining: progress.estimatedRemaining || 0,
      };
      migrationVisible.value = true;

      if (progress.stage === 'done') {
        migrationCloseTimer = setTimeout(() => {
          migrationVisible.value = false;
          migrationCloseTimer = null;
        }, 1200);
      }
    });
  });

  onUnmounted(() => {
    if (migrationProgressOff) {
      migrationProgressOff();
      migrationProgressOff = null;
    }
    if (migrationCloseTimer) {
      clearTimeout(migrationCloseTimer);
    }
  });

  return {
    migrationVisible,
    migrationProgress,
    formatRemainingTime: formatMigrationRemainingTime,
  };
};
