<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import { t } from '../../i18n';

defineProps<{
  sdk: SdkInfo;
  copiedPath: string | null;
  expandedVersions: Record<string, boolean>;
  displayVersion: (version?: string) => string;
}>();

const emit = defineEmits(['copy-path', 'toggle-expand']);
</script>

<template>
  <div class="vfox-versions-section">
    <h2>{{ t('sdk.nonevfox.title') }}</h2>
    <div class="version-card">
      <div class="version-card-header">
        <div class="flex-align-center flex-gap-12" style="min-width: 0; flex: 1;">
          <h3
            class="version-title"
            :style="{
              display: 'flex',
              alignItems: expandedVersions['sys-' + sdk.path] ? 'flex-start' : 'center',
              flexWrap: expandedVersions['sys-' + sdk.path] ? 'wrap' : 'nowrap',
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
                whiteSpace: expandedVersions['sys-' + sdk.path] ? 'normal' : 'nowrap'
              }"
            >
              {{ displayVersion(sdk.versions?.[0]?.version) }}
            </span>
            <button
              v-if="displayVersion(sdk.versions?.[0]?.version).length > 50"
              class="btn text small"
              style="padding: 0 4px; min-width: auto; height: 20px; line-height: 20px; flex-shrink: 0;"
              @click="emit('toggle-expand', 'sys-' + sdk.path)"
            >
              {{ expandedVersions['sys-' + sdk.path] ? t('common.collapse') : t('common.expand') }}
            </button>
          </h3>
          <span class="system-tag" style="flex-shrink: 0;">{{ t('sdk.custom') }}</span>
        </div>
      </div>
      <div class="version-card-body">
        <div class="path-label">{{ t('sdk.exe_path') }}</div>
        <div class="path-box">
          <code class="path-text">{{ sdk.path }}</code>
          <button class="btn icon-btn" :title="t('common.copy_path')" @click="emit('copy-path', sdk.path)">
            <svg
              v-if="copiedPath === sdk.path"
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
        <p class="empty-state" style="margin-top: 12px; font-size: 14px; text-align: left;">
          {{ t('sdk.system.manage_hint') }}
        </p>
      </div>
    </div>
  </div>
</template>
