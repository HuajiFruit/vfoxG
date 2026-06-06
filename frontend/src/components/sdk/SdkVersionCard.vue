<script lang="ts" setup>
import { t } from '../../i18n';
import { UnifiedSdkVersion } from './ManagedSdkDetailView.vue';

defineProps<{
  sdkName: string;
  version: UnifiedSdkVersion;
  versionKey: string;
  isExpanded: boolean;
  copiedPath: string | null;
  usingVersion: string | null;
  unusingSdk: string | null;
}>();

const emit = defineEmits([
  'toggle-expand',
  'copy-path',
  'use-custom-path',
  'use-version',
  'unuse',
  'remove-custom-path',
  'uninstall-version',
]);
</script>

<template>
  <div class="version-card" :class="{ 'is-current': version.isCurrent }">
    <div class="version-card-header">
      <div class="flex-align-center flex-gap-12" style="min-width: 0; flex: 1;">
        <h3
          class="version-title"
          :style="{
            display: 'flex',
            alignItems: isExpanded ? 'flex-start' : 'center',
            flexWrap: isExpanded ? 'wrap' : 'nowrap',
            gap: '8px',
            minWidth: 0,
            flex: '0 1 auto'
          }"
        >
          <span
            :style="{
              wordBreak: 'break-all',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: isExpanded ? 'normal' : 'nowrap'
            }"
          >
            {{ version.version }}
          </span>
          <button
            v-if="version.version.length > 50"
            class="btn text small"
            style="padding: 0 4px; min-width: auto; height: 20px; line-height: 20px; flex-shrink: 0;"
            @click="emit('toggle-expand', versionKey)"
          >
            {{ isExpanded ? t('common.collapse') : t('common.expand') }}
          </button>
        </h3>
        <span v-if="version.isCurrent" class="current-tag" style="flex-shrink: 0;">{{ t('sdk.current') }}</span>
        <span v-if="version.isCustom" class="system-tag" style="flex-shrink: 0;">{{ t('sdk.custom') }}</span>
        <span v-else class="vfox-tag" style="flex-shrink: 0;">vfox</span>
      </div>
      <div class="version-actions">
        <button
          v-if="!version.isCurrent"
          class="btn tonal small"
          :disabled="usingVersion === (version.isCustom ? version.path : version.version)"
          @click="version.isCustom
            ? emit('use-custom-path', sdkName, version.path)
            : emit('use-version', sdkName, version.vfoxVersion)"
        >
          {{ usingVersion === (version.isCustom ? version.path : version.version) ? '...' : t('sdk.use') }}
        </button>
        <button
          v-if="version.isCurrent"
          class="btn text small danger"
          :disabled="unusingSdk === sdkName"
          @click="emit('unuse', sdkName)"
        >
          {{ unusingSdk === sdkName ? '...' : t('sdk.unset') }}
        </button>
        <button v-if="version.isCustom" class="btn text small danger" @click="emit('remove-custom-path', sdkName, version.path)">
          {{ t('sdk.remove') }}
        </button>
        <button v-else class="btn text small danger" @click="emit('uninstall-version', sdkName, version.vfoxVersion)">
          {{ t('sdk.uninstall') }}
        </button>
      </div>
    </div>
    <div class="version-card-body">
      <div class="path-label">{{ version.isCustom ? t('sdk.exe_path') : t('sdk.install_path') }}</div>
      <div v-if="version.path" class="path-box">
        <code class="path-text">{{ version.path }}</code>
        <button class="btn icon-btn" :title="t('common.copy_path')" @click="emit('copy-path', version.path)">
          <svg
            v-if="copiedPath === version.path"
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--md-primary)"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
          </svg>
        </button>
      </div>
      <div v-else class="path-box loading-path">
        <div class="spinner small"></div>
        {{ t('sdk.loading_path') }}
      </div>
    </div>
  </div>
</template>
