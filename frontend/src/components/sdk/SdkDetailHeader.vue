<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import { t } from '../../i18n';
import PluginIcon from '../plugin/PluginIcon.vue';

defineProps<{
  sdk: SdkInfo;
  checkingCompat: boolean;
  isPathOverrideApplied: boolean;
  hijacking: boolean;
  restoring: boolean;
  removingPlugin: string | null;
  pathOverrideTooltip: string;
  pathOverrideRemoveTooltip: string;
}>();

const emit = defineEmits(['back', 'enable-path-override', 'disable-path-override', 'remove-plugin']);
</script>

<template>
  <div class="detail-header">
    <button class="btn tonal small back-btn" @click="emit('back')">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M19 12H5M12 19l-7-7 7-7" />
      </svg>
      {{ t('sdk.back') }}
    </button>

    <div class="detail-title-area" style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
      <div style="display: flex; align-items: center; gap: 24px;">
        <PluginIcon :name="sdk.name" class="sdk-icon-huge" />
        <div class="detail-title-text">
          <h1>{{ sdk.name }}</h1>
        </div>
      </div>

      <div v-if="sdk.source === 'vfox'" style="display: flex; gap: 8px; align-items: flex-start;">
        <div style="display: flex; flex-direction: column; align-items: center; gap: 4px;">
          <button
            v-if="checkingCompat"
            class="btn tonal small"
            disabled
            style="min-width: 140px; display: flex; justify-content: center; align-items: center; background: transparent;"
          >
            <div
              class="spinner small-spinner"
              style="width: 16px; height: 16px; border-width: 2px; border-color: var(--md-outline) transparent var(--md-outline) transparent;"
            ></div>
          </button>
          <button
            v-else-if="!isPathOverrideApplied"
            class="btn tonal small"
            :disabled="hijacking || restoring"
            style="min-width: 140px; display: flex; justify-content: center; align-items: center;"
            @click="emit('enable-path-override', sdk.name)"
          >
            <div v-if="hijacking" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px;"></div>
            <template v-else>
              <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">security</span>
              {{ t('sdk.path_override.enable') }}
              <div class="custom-tooltip-container" style="margin-left: 6px; display: flex;">
                <span class="material-symbols-outlined" style="font-size: 14px; color: var(--md-outline); cursor: help;">info</span>
                <div class="custom-tooltip-content">
                  {{ pathOverrideTooltip }}
                </div>
              </div>
            </template>
          </button>
          <button
            v-else
            class="btn outlined small"
            :disabled="hijacking || restoring"
            style="min-width: 140px; display: flex; justify-content: center; align-items: center;"
            @click="emit('disable-path-override', sdk.name)"
          >
            <div v-if="restoring" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px;"></div>
            <template v-else>
              <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">restore</span>
              {{ t('sdk.path_override.disable') }}
              <div class="custom-tooltip-container" style="margin-left: 6px; display: flex;">
                <span class="material-symbols-outlined" style="font-size: 14px; color: var(--md-outline); cursor: help;">info</span>
                <div class="custom-tooltip-content">
                  {{ pathOverrideRemoveTooltip }}
                </div>
              </div>
            </template>
          </button>
          <span v-if="!checkingCompat && !isPathOverrideApplied" style="font-size: 11px; color: var(--md-outline); opacity: 0.8;">
            {{ t('sdk.path_override.hint') }}
          </span>
        </div>

        <button
          class="btn tonal small"
          style="color: var(--md-error); background: rgba(239, 68, 68, 0.1); min-width: 120px; display: flex; justify-content: center; align-items: center;"
          :disabled="removingPlugin === sdk.name"
          @click="emit('remove-plugin', sdk.name)"
        >
          <div
            v-if="removingPlugin === sdk.name"
            class="spinner small-spinner"
            style="width: 16px; height: 16px; border-width: 2px; border-top-color: var(--md-error);"
          ></div>
          <template v-else>
            <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">delete</span>
            {{ t('sdk.remove_plugin') }}
          </template>
        </button>
      </div>
    </div>
  </div>
</template>
