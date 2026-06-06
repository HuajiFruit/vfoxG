<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import { t } from '../../i18n';
import SdkVersionCard from './SdkVersionCard.vue';

export interface UnifiedSdkVersion {
  isCustom: boolean;
  version: string;
  path: string;
  isCurrent: boolean;
  sysSdk?: SdkInfo;
  vfoxVersion?: string;
}

defineProps<{
  sdk: SdkInfo;
  hasDetail: boolean;
  hasDetailError: boolean;
  unifiedVersions: UnifiedSdkVersion[];
  expandedVersions: Record<string, boolean>;
  copiedPath: string | null;
  usingVersion: string | null;
  unusingSdk: string | null;
  searchingFor: string | null;
  searchLoading: boolean;
  filteredSearchResults: string[];
  searchQuery: string;
  isAddingPathMode: string | null;
  customPathInput: string;
  customVersionInput: string;
  isAddingCustomPath: boolean;
  isSearchVersionInstalled: (name: string, version: string) => boolean;
}>();

const emit = defineEmits([
  'toggle-expand',
  'copy-path',
  'use-custom-path',
  'use-version',
  'unuse',
  'remove-custom-path',
  'uninstall-version',
  'search',
  'install',
  'update:searchQuery',
  'cancel-search',
  'start-add-custom',
  'cancel-add-custom',
  'update:customPathInput',
  'update:customVersionInput',
  'detect-version',
  'add-custom-path',
]);

const versionKey = (version: UnifiedSdkVersion) =>
  version.isCustom ? 'sys-' + version.path : 'vfox-' + version.version;
</script>

<template>
  <div class="vfox-versions-section">
    <h2>{{ t('sdk.version_list') }}</h2>

    <div v-if="!hasDetail && !hasDetailError" class="flex-center" style="padding: 20px;">
      <div class="spinner"></div>
    </div>

    <div v-else-if="unifiedVersions.length" class="versions-grid">
      <SdkVersionCard
        v-for="ver in unifiedVersions"
        :key="versionKey(ver)"
        :sdk-name="sdk.name"
        :version="ver"
        :version-key="versionKey(ver)"
        :is-expanded="Boolean(expandedVersions[versionKey(ver)])"
        :copied-path="copiedPath"
        :using-version="usingVersion"
        :unusing-sdk="unusingSdk"
        @toggle-expand="emit('toggle-expand', $event)"
        @copy-path="emit('copy-path', $event)"
        @use-custom-path="(...args) => emit('use-custom-path', ...args)"
        @use-version="(...args) => emit('use-version', ...args)"
        @unuse="emit('unuse', $event)"
        @remove-custom-path="(...args) => emit('remove-custom-path', ...args)"
        @uninstall-version="(...args) => emit('uninstall-version', ...args)"
      />
    </div>

    <div v-else class="empty-state">{{ t('sdk.no_versions_installed') }}</div>

    <div v-if="sdk.source === 'vfox'" class="install-section-large" style="margin-top: 24px; padding-top: 24px; border-top: 1px dashed var(--panel-border, var(--md-outline-variant));">
      <button v-if="searchingFor !== sdk.name" class="btn primary large-btn" @click="emit('search', sdk.name)">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        {{ t('sdk.install_new') }}
      </button>

      <div v-else class="search-box large">
        <div class="search-header">
          <input
            :value="searchQuery"
            type="text"
            class="search-input"
            :placeholder="t('sdk.search_versions.placeholder')"
            autofocus
            @input="emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
          >
          <button class="btn text" @click="emit('cancel-search')">{{ t('common.cancel') }}</button>
        </div>
        <div v-if="searchLoading" class="flex-center" style="padding: 24px;">
          <div class="spinner"></div>
        </div>
        <div v-else class="search-results-grid">
          <button
            v-for="ver in filteredSearchResults"
            :key="ver"
            class="search-result-card"
            :class="{ installed: isSearchVersionInstalled(sdk.name, ver) }"
            :disabled="isSearchVersionInstalled(sdk.name, ver)"
            @click="emit('install', sdk.name, ver)"
          >
            <span>{{ ver }}</span>
            <span v-if="isSearchVersionInstalled(sdk.name, ver)" class="installed-text">{{ t('market.installed') }}</span>
            <span v-else class="install-text">{{ t('market.install') }}</span>
          </button>
          <div v-if="!filteredSearchResults.length" class="empty-state" style="grid-column: 1/-1;">
            {{ t('sdk.no_matching_versions') }}
          </div>
        </div>
      </div>
    </div>

    <div class="install-section-large" style="margin-top: 24px; padding-top: 24px; border-top: 1px dashed var(--panel-border, var(--md-outline-variant));">
      <button v-if="isAddingPathMode !== sdk.name" class="btn tonal small" @click="emit('start-add-custom', sdk.name)">
        + {{ t('sdk.add_custom') }}
      </button>
      <div v-else class="search-box large">
        <div class="search-header">
          <input
            :value="customPathInput"
            type="text"
            class="search-input"
            :placeholder="t('sdk.custom_path.placeholder')"
            style="font-size: 14px; padding: 12px 16px; flex: 2; box-sizing: border-box; margin: 0; height: 100%; min-height: 44px;"
            autofocus
            @input="emit('update:customPathInput', ($event.target as HTMLInputElement).value)"
            @blur="emit('detect-version', sdk.name)"
          >
          <input
            :value="customVersionInput"
            type="text"
            class="search-input"
            :placeholder="t('sdk.version.placeholder')"
            style="font-size: 14px; padding: 12px 16px; flex: 1; box-sizing: border-box; margin: 0; height: 100%;"
            @input="emit('update:customVersionInput', ($event.target as HTMLInputElement).value)"
          >
          <button class="btn text" @click="emit('cancel-add-custom')">{{ t('sdk.cancel') }}</button>
          <button
            class="btn primary"
            :disabled="isAddingCustomPath || !customPathInput.trim()"
            style="min-width: 80px; display: flex; justify-content: center;"
            @click="emit('add-custom-path', sdk.name)"
          >
            <div v-if="isAddingCustomPath" class="spinner small-spinner" style="width: 14px; height: 14px; border-width: 2px;"></div>
            <template v-else>{{ t('sdk.add') }}</template>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
