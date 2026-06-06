<script lang="ts" setup>
import { t } from '../../i18n';

defineProps<{
  showSteps?: boolean;
}>();

const emit = defineEmits(['open-plugin-market', 'open-sync']);
</script>

<template>
  <div class="sdk-empty-panel">
    <div class="sdk-empty-art" aria-hidden="true">
      <span class="material-symbols-outlined">deployed_code</span>
    </div>
    <div class="sdk-empty-content">
      <p class="empty-kicker">{{ t('sdk.empty.kicker') }}</p>
      <h3>{{ t('sdk.empty.title') }}</h3>
      <p>{{ t('sdk.empty.desc') }}</p>
    </div>
    <div class="sdk-empty-actions">
      <button class="btn primary" @click="emit('open-plugin-market')">
        <span class="material-symbols-outlined">extension</span>
        {{ t('sdk.empty.market') }}
      </button>
      <button class="btn tonal" @click="emit('open-sync')">
        <span class="material-symbols-outlined">upload_file</span>
        {{ t('sdk.empty.import') }}
      </button>
    </div>
    <div v-if="showSteps" class="sdk-empty-steps">
      <div class="sdk-empty-step">
        <span class="material-symbols-outlined">add_circle</span>
        <span>{{ t('sdk.empty.step.plugin') }}</span>
      </div>
      <div class="sdk-empty-step">
        <span class="material-symbols-outlined">download</span>
        <span>{{ t('sdk.empty.step.install') }}</span>
      </div>
      <div class="sdk-empty-step">
        <span class="material-symbols-outlined">alt_route</span>
        <span>{{ t('sdk.empty.step.use') }}</span>
      </div>
    </div>
  </div>
</template>
