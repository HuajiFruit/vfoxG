<script lang="ts" setup>
import { computed } from 'vue';
import { t } from '../../i18n';
import type { MigrationProgress } from '../../composables/useMigrationProgress';

const props = defineProps<{
  visible: boolean;
  progress: MigrationProgress;
  formatRemainingTime: (seconds: number) => string;
}>();

const progressIcon = computed(() => {
  if (props.progress.stage === 'done') return 'check_circle';
  if (props.progress.stage === 'error') return 'error';
  return 'drive_folder_upload';
});

const progressTitle = computed(() => {
  if (props.progress.stage === 'done') return t('migration.done');
  if (props.progress.stage === 'preparing') return t('migration.preparing');
  return t('migration.copying');
});
</script>

<template>
  <Transition name="migration-fade">
    <div v-if="visible" class="migration-overlay" role="status" aria-live="polite">
      <div class="migration-dialog">
        <div class="migration-icon">
          <span class="material-symbols-outlined">{{ progressIcon }}</span>
        </div>
        <div class="migration-content">
          <p class="migration-kicker">{{ t('migration.kicker') }}</p>
          <h3>{{ progressTitle }}</h3>
          <div class="migration-current">
            <span>{{ t('migration.current') }}</span>
            <code>{{ progress.current || t('migration.scanning') }}</code>
          </div>
          <div class="migration-progress-bar">
            <div class="migration-progress-fill" :style="{ width: `${progress.percent}%` }"></div>
          </div>
          <div class="migration-meta">
            <span>{{ progress.completed }} / {{ progress.total }}</span>
            <span>{{ progress.percent }}%</span>
            <span>{{ formatRemainingTime(progress.estimatedRemaining) }}</span>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>
