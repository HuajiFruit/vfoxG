<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import { t } from '../../i18n';
import SdkDetailView from './SdkDetailView.vue';
import SdkEmptyState from './SdkEmptyState.vue';
import SdkListView from './SdkListView.vue';
import SdkRemovalModals from './SdkRemovalModals.vue';
import type { UnifiedSdkVersion } from './ManagedSdkDetailView.vue';

type ConfirmAction = {
  type: 'removePlugin' | 'uninstallVersion' | 'removeCustomSdk' | null;
  name: string;
  version?: string;
  path?: string;
};

defineProps<{
  activeView: 'list' | 'detail';
  transitionName: string;
  loading: boolean;
  sdks: SdkInfo[];
  vfoxSdks: SdkInfo[];
  systemSdks: SdkInfo[];
  selectedSdk: SdkInfo | null;
  checkingCompat: boolean;
  isPathOverrideApplied: boolean;
  hijacking: boolean;
  restoring: boolean;
  removingPlugin: string | null;
  pathOverrideTooltip: string;
  pathOverrideRemoveTooltip: string;
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
  confirmAction: ConfirmAction;
  removePluginName: string | null;
  removePluginCustomSdks: Array<{ path: string; version: string }>;
  getTotalVersionCount: (sdk: SdkInfo) => number;
  displayVersion: (version?: string) => string;
  truncateVersion: (version?: string) => string;
  isSearchVersionInstalled: (sdkName: string, version: string) => boolean;
}>();

const emit = defineEmits([
  'open-detail',
  'open-plugin-market',
  'open-sync',
  'back',
  'enable-path-override',
  'disable-path-override',
  'remove-plugin',
  'toggle-expand',
  'copy-path',
  'use-custom-path',
  'use-version',
  'unuse',
  'remove-custom-path',
  'uninstall-version',
  'search',
  'install',
  'cancel-search',
  'start-add-custom',
  'cancel-add-custom',
  'detect-version',
  'add-custom-path',
  'update:searchQuery',
  'update:customPathInput',
  'update:customVersionInput',
  'confirm-action',
  'cancel-action',
  'confirm-remove-plugin',
  'cancel-remove-plugin',
]);
</script>

<template>
  <div class="sdk-manager">
    <Transition :name="transitionName" mode="out-in">
      <SdkListView
        v-if="activeView === 'list'"
        key="list"
        :loading="loading"
        :sdks="sdks"
        :vfox-sdks="vfoxSdks"
        :system-sdks="systemSdks"
        :get-total-version-count="getTotalVersionCount"
        :display-version="displayVersion"
        :truncate-version="truncateVersion"
        @open-detail="emit('open-detail', $event)"
        @open-plugin-market="emit('open-plugin-market')"
        @open-sync="emit('open-sync')"
      />

      <SdkDetailView
        v-else-if="activeView === 'detail' && selectedSdk"
        key="detail"
        :sdk="selectedSdk"
        :checking-compat="checkingCompat"
        :is-path-override-applied="isPathOverrideApplied"
        :hijacking="hijacking"
        :restoring="restoring"
        :removing-plugin="removingPlugin"
        :path-override-tooltip="pathOverrideTooltip"
        :path-override-remove-tooltip="pathOverrideRemoveTooltip"
        :has-detail="hasDetail"
        :has-detail-error="hasDetailError"
        :unified-versions="unifiedVersions"
        :expanded-versions="expandedVersions"
        :copied-path="copiedPath"
        :using-version="usingVersion"
        :unusing-sdk="unusingSdk"
        :searching-for="searchingFor"
        :search-loading="searchLoading"
        :filtered-search-results="filteredSearchResults"
        :search-query="searchQuery"
        :is-adding-path-mode="isAddingPathMode"
        :custom-path-input="customPathInput"
        :custom-version-input="customVersionInput"
        :is-adding-custom-path="isAddingCustomPath"
        :display-version="displayVersion"
        :is-search-version-installed="isSearchVersionInstalled"
        @back="emit('back')"
        @enable-path-override="emit('enable-path-override', $event)"
        @disable-path-override="emit('disable-path-override', $event)"
        @remove-plugin="emit('remove-plugin', $event)"
        @toggle-expand="emit('toggle-expand', $event)"
        @copy-path="emit('copy-path', $event)"
        @use-custom-path="(...args) => emit('use-custom-path', ...args)"
        @use-version="(...args) => emit('use-version', ...args)"
        @unuse="emit('unuse', $event)"
        @remove-custom-path="(...args) => emit('remove-custom-path', ...args)"
        @uninstall-version="(...args) => emit('uninstall-version', ...args)"
        @search="emit('search', $event)"
        @install="(...args) => emit('install', ...args)"
        @cancel-search="emit('cancel-search')"
        @start-add-custom="emit('start-add-custom', $event)"
        @cancel-add-custom="emit('cancel-add-custom')"
        @detect-version="emit('detect-version', $event)"
        @add-custom-path="emit('add-custom-path', $event)"
        @update:search-query="emit('update:searchQuery', $event)"
        @update:custom-path-input="emit('update:customPathInput', $event)"
        @update:custom-version-input="emit('update:customVersionInput', $event)"
      />

      <div v-else key="empty-list" class="view-container">
        <div class="sdk-list-header">
          <h2>{{ t('sdk.installed.title') }}</h2>
        </div>
        <SdkEmptyState @open-plugin-market="emit('open-plugin-market')" @open-sync="emit('open-sync')" />
      </div>
    </Transition>

    <SdkRemovalModals
      :confirm-action="confirmAction"
      :remove-plugin-name="removePluginName"
      :remove-plugin-custom-sdks="removePluginCustomSdks"
      @confirm-action="emit('confirm-action')"
      @cancel-action="emit('cancel-action')"
      @confirm-remove-plugin="emit('confirm-remove-plugin', $event)"
      @cancel-remove-plugin="emit('cancel-remove-plugin')"
    />
  </div>
</template>
