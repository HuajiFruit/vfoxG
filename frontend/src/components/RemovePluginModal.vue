<script lang="ts" setup>
import { ref } from 'vue';
import { t } from '../i18n';

const props = defineProps<{
  pluginName: string;
  customSdks: Array<{ path: string; version: string }>;
}>();

const emit = defineEmits(['confirm', 'cancel']);

// null = clean removal (no SDK kept), string = chosen path to restore
const selectedPath = ref<string | null>(null);

const handleConfirm = () => {
  emit('confirm', selectedPath.value);
};
</script>

<template>
  <Transition name="modal" appear>
    <div class="modal-overlay" @click="emit('cancel')">
      <div class="modal-content remove-plugin-modal" @click.stop>
        <h2 class="modal-title">{{ t('sdk.remove_plugin') }}</h2>
        
        <p class="modal-message" style="margin-bottom: 8px;">
          {{ t('sdk.confirm.remove_plugin') }} '{{ pluginName }}'?
        </p>

        <div v-if="customSdks.length > 0" class="sdk-choice-section">
          <p class="sdk-choice-hint">
            {{ t('sdk.remove_plugin.choose_sdk') }}
          </p>

          <!-- Clean removal option -->
          <label class="sdk-choice-item" :class="{ active: selectedPath === null }">
            <input type="radio" name="sdk-choice" :checked="selectedPath === null" @change="selectedPath = null" />
            <div class="sdk-choice-info">
              <span class="sdk-choice-label">{{ t('sdk.remove_plugin.clean') }}</span>
              <span class="sdk-choice-desc">{{ t('sdk.remove_plugin.clean_desc') }}</span>
            </div>
          </label>

          <!-- Custom SDK options -->
          <label 
            v-for="sdk in customSdks" 
            :key="sdk.path" 
            class="sdk-choice-item" 
            :class="{ active: selectedPath === sdk.path }"
          >
            <input type="radio" name="sdk-choice" :checked="selectedPath === sdk.path" @change="selectedPath = sdk.path" />
            <div class="sdk-choice-info">
              <span class="sdk-choice-label">{{ sdk.version || 'unknown' }}</span>
              <code class="sdk-choice-path">{{ sdk.path }}</code>
            </div>
          </label>
        </div>

        <div class="modal-actions">
          <button class="btn tonal" @click="emit('cancel')">{{ t('sdk.cancel') }}</button>
          <button class="btn primary" style="background: var(--md-error);" @click="handleConfirm">{{ t('sdk.remove') }}</button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.remove-plugin-modal {
  max-width: 520px;
  width: 90vw;
}

.sdk-choice-section {
  margin: 16px 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sdk-choice-hint {
  color: var(--text-secondary);
  font-size: 13px;
  margin: 0 0 8px 0;
  line-height: 1.5;
}

.sdk-choice-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container);
  cursor: pointer;
  transition: all 0.2s ease;
  border: 2px solid transparent;
}

.sdk-choice-item:hover {
  background: var(--md-sys-color-surface-container-high);
}

.sdk-choice-item.active {
  border-color: var(--md-primary);
  background: color-mix(in srgb, var(--md-primary) 8%, var(--md-sys-color-surface-container));
}

.sdk-choice-item input[type="radio"] {
  margin-top: 3px;
  accent-color: var(--md-primary);
  flex-shrink: 0;
  width: 16px;
  height: 16px;
}

.sdk-choice-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.sdk-choice-label {
  font-weight: 600;
  font-size: 14px;
  color: var(--md-sys-color-on-surface);
}

.sdk-choice-path {
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-all;
  background: var(--md-sys-color-surface-container-lowest);
  padding: 4px 8px;
  border-radius: 6px;
}

.sdk-choice-desc {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
}
</style>
