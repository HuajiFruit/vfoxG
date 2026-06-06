<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import SdkDetailHeader from './SdkDetailHeader.vue';
import ManagedSdkDetailView, { UnifiedSdkVersion } from './ManagedSdkDetailView.vue';
import SystemSdkDetailView from './SystemSdkDetailView.vue';

defineProps<{
  sdk: SdkInfo;
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
  displayVersion: (version?: string) => string;
  isSearchVersionInstalled: (sdkName: string, version: string) => boolean;
}>();

const emit = defineEmits([
  'back',
  'enable-path-override',
  'disable-path-override',
  'remove-plugin',
  'copy-path',
  'toggle-expand',
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
</script>

<template>
  <div class="view-container detail-view">
    <SdkDetailHeader
      :sdk="sdk"
      :checking-compat="checkingCompat"
      :is-path-override-applied="isPathOverrideApplied"
      :hijacking="hijacking"
      :restoring="restoring"
      :removing-plugin="removingPlugin"
      :path-override-tooltip="pathOverrideTooltip"
      :path-override-remove-tooltip="pathOverrideRemoveTooltip"
      @back="emit('back')"
      @enable-path-override="emit('enable-path-override', $event)"
      @disable-path-override="emit('disable-path-override', $event)"
      @remove-plugin="emit('remove-plugin', $event)"
    />

    <div class="detail-body">
      <SystemSdkDetailView
        v-if="sdk.source === 'system'"
        :sdk="sdk"
        :copied-path="copiedPath"
        :expanded-versions="expandedVersions"
        :display-version="displayVersion"
        @copy-path="emit('copy-path', $event)"
        @toggle-expand="emit('toggle-expand', $event)"
      />

      <ManagedSdkDetailView
        v-else
        :search-query="searchQuery"
        :custom-path-input="customPathInput"
        :custom-version-input="customVersionInput"
        :sdk="sdk"
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
        :is-adding-path-mode="isAddingPathMode"
        :is-adding-custom-path="isAddingCustomPath"
        :is-search-version-installed="isSearchVersionInstalled"
        @toggle-expand="emit('toggle-expand', $event)"
        @copy-path="emit('copy-path', $event)"
        @use-custom-path="(...args) => emit('use-custom-path', ...args)"
        @use-version="(...args) => emit('use-version', ...args)"
        @unuse="emit('unuse', $event)"
        @remove-custom-path="(...args) => emit('remove-custom-path', ...args)"
        @uninstall-version="(...args) => emit('uninstall-version', ...args)"
        @search="emit('search', $event)"
        @install="(...args) => emit('install', ...args)"
        @update:search-query="emit('update:searchQuery', $event)"
        @cancel-search="emit('cancel-search')"
        @start-add-custom="emit('start-add-custom', $event)"
        @cancel-add-custom="emit('cancel-add-custom')"
        @update:custom-path-input="emit('update:customPathInput', $event)"
        @update:custom-version-input="emit('update:customVersionInput', $event)"
        @detect-version="emit('detect-version', $event)"
        @add-custom-path="emit('add-custom-path', $event)"
      />
    </div>
  </div>
</template>
