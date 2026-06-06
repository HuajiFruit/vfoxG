<script lang="ts" setup>
import { computed, type CSSProperties } from 'vue';
import { t } from '../../i18n';
import type { TaskPhase, ToastStatus } from '../../composables/useTaskTerminal';

const props = defineProps<{
  visible: boolean;
  status: ToastStatus;
  phase: TaskPhase;
  title: string;
  message: string;
  showProgress: boolean;
  isDeterminateDownloadProgress: boolean;
  progress: number;
  downloadSpeed: string;
  progressStyle?: CSSProperties;
  busyHintVisible: boolean;
}>();

const phaseIcon = computed(() => {
  if (props.phase === 'download') return 'download';
  if (props.phase === 'install') return 'inventory_2';
  return 'hourglass_top';
});

const phaseLabel = computed(() => props.phase === 'download'
  ? t('toast.phase.download')
  : t('toast.phase.install'));
</script>

<template>
  <Transition name="toast-slide">
    <div v-if="visible" class="task-toast" :class="`toast-${status}`">
      <div class="toast-icon">
        <div v-if="status === 'running'" class="toast-phase-icon" :class="`phase-${phase}`">
          <span class="material-symbols-outlined">{{ phaseIcon }}</span>
        </div>
        <svg v-else-if="status === 'success'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
        <svg v-else-if="status === 'error'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
        <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
      </div>
      <div class="toast-content">
        <div class="toast-title-row">
          <div class="toast-title">{{ title }}</div>
          <div v-if="status === 'running' && phase !== 'default'" class="toast-phase-pill" :class="`phase-${phase}`">
            {{ phaseLabel }}
          </div>
        </div>
        <div class="toast-subtitle" :class="{'text-error': status === 'error', 'text-success': status === 'success'}">
          {{ message }}
        </div>
        <div
          v-if="showProgress"
          class="toast-progress"
          :class="{ indeterminate: status === 'running' && !isDeterminateDownloadProgress, error: status === 'error', success: status === 'success', download: phase === 'download', install: phase === 'install' }"
        >
          <div class="toast-progress-fill" :style="progressStyle"></div>
        </div>
        <div v-if="isDeterminateDownloadProgress" class="toast-progress-meta">
          <span class="toast-download-speed">{{ downloadSpeed }}</span>
          <span class="toast-progress-label">{{ progress }}%</span>
        </div>
        <Transition name="busy-hint">
          <div v-if="busyHintVisible" class="toast-busy-hint">{{ t('toast.please_wait') }}</div>
        </Transition>
      </div>
    </div>
  </Transition>
</template>
