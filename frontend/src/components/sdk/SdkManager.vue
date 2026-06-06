<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import { computed, ref, inject } from 'vue';
import { t } from '../../i18n';
import SdkManagerShell from './SdkManagerShell.vue';
import { useCopyablePath } from '../../composables/useCopyablePath';
import { useExpandableVersions } from '../../composables/useExpandableVersions';
import { usePathOverrideActions } from '../../composables/usePathOverrideActions';
import { useNotifyBridge } from '../../composables/useNotifyBridge';
import { useSdkDetailData } from '../../composables/useSdkDetailData';
import { useSdkDetailNavigation } from '../../composables/useSdkDetailNavigation';
import { useSdkInventory } from '../../composables/useSdkInventory';
import { useSdkManagerLifecycle } from '../../composables/useSdkManagerLifecycle';
import { useSdkRemovalDialogs } from '../../composables/useSdkRemovalDialogs';
import { useSdkTaskRunner, type RunTerminalTask } from '../../composables/useSdkTaskRunner';
import { useSdkVersionActions } from '../../composables/useSdkVersionActions';
import { useCustomSdkPaths } from '../../composables/useCustomSdkPaths';
import { useUnifiedSdkVersions } from '../../composables/useUnifiedSdkVersions';
import { displayVersion, truncateVersion } from '../../composables/sdkVersionFormat';
import { useSdkVersionSearch } from '../../composables/useSdkVersionSearch';

type SidebarAction = { id: number; type: 'display' };
const props = defineProps<{ sidebarAction?: SidebarAction | null }>();
const emit = defineEmits(['start-task', 'notify', 'sidebar-action-done', 'open-plugin-market', 'open-sync']);
const runTerminalTask = inject<RunTerminalTask>('runTerminalTask');
const selectedSdk = ref<SdkInfo | null>(null);

const {
  notifyError,
  notifySuccess,
  notifyInfo,
  getErrorMessage,
} = useNotifyBridge({
  emitNotify: payload => emit('notify', payload),
});

const {
  runTask,
  notifyTaskError,
} = useSdkTaskRunner({
  runTerminalTask,
  emitStartTask: (title: string) => emit('start-task', title),
  notifyInfo,
  notifyError,
  formatError: getErrorMessage,
});

const {
  sdkDetails,
  detailError,
  versionPaths,
  beginDetailRequest,
  invalidateDetailRequests,
  isCurrentDetailRequest,
  fetchDetail,
  refreshSdkDetail,
  fetchDetailVersionPaths,
  fetchSingleVersionPath,
  clearSdkCurrentVersion,
  removeVersionPath,
} = useSdkDetailData({
  selectedSdk,
  notifyError,
  formatError: getErrorMessage,
});

const {
  checkingCompat,
  isPathOverrideApplied,
  activeCustomSdk,
  hijacking,
  restoring,
  pathOverrideTooltip,
  pathOverrideRemoveTooltip,
  invalidateCompatRequests,
  clearActiveCustomSdk,
  loadPlatformInfo,
  checkCompatMode,
  handleHijackPlugin,
  handleRestorePlugin,
} = usePathOverrideActions({
  selectedSdk,
  notifyError,
  notifySuccess,
  formatError: getErrorMessage,
});

const {
  sdks,
  loading,
  nonVfoxSdksMap,
  vfoxSdks,
  systemSdks,
  fetchAllSdks,
  fetchVfoxSdks,
  refreshNonVfoxSdks,
} = useSdkInventory({
  notifyError,
  formatError: getErrorMessage,
  sdkRefreshError: t('sdk.refresh_error'),
  sdkLoadError: t('sdk.load_error'),
  onSdkListChanged: () => {
    if (selectedSdk.value) {
      refreshSdkDetail(selectedSdk.value.name);
      checkCompatMode(selectedSdk.value.name);
    }
  },
});

const {
  searchingFor,
  searchLoading,
  searchQuery,
  filteredSearchResults,
  isSearchVersionInstalled,
  handleSearch,
  cancelSearch,
  resetSearch,
} = useSdkVersionSearch({
  sdkDetails,
  notifyError,
  formatError: getErrorMessage,
});

const {
  activeView,
  transitionName,
  openDetail,
  closeDetail,
} = useSdkDetailNavigation({
  selectedSdk,
  notifyError,
  formatError: getErrorMessage,
  beginDetailRequest,
  invalidateDetailRequests,
  invalidateCompatRequests,
  isCurrentDetailRequest,
  fetchDetail,
  fetchDetailVersionPaths,
  refreshNonVfoxSdks,
  checkCompatMode,
  resetSearch,
});

const { unifiedVersions } = useUnifiedSdkVersions({
  selectedSdk,
  sdkDetails,
  versionPaths,
  nonVfoxSdksMap,
  activeCustomSdk,
});

const {
  usingVersion,
  unusingSdk,
  handleUseCustomPath,
  handleUse,
  handleUnuse,
  handleInstall,
  handleUninstall,
  removeCurrentCustomSdkBeforeDelete,
} = useSdkVersionActions({
  selectedSdk,
  sdkDetails,
  activeCustomSdk,
  runTask,
  notifyError,
  notifyTaskError,
  formatError: getErrorMessage,
  refreshSdkDetail,
  fetchVfoxSdks,
  checkCompatMode,
  clearSdkCurrentVersion,
  clearActiveCustomSdk,
  fetchSingleVersionPath,
  removeVersionPath,
});

const getTotalVersionCount = (sdk: SdkInfo) =>
  (sdk.versions?.length || 0) + (nonVfoxSdksMap.value[sdk.name]?.length || 0);

const {
  customPathInput,
  customVersionInput,
  isAddingCustomPath,
  isAddingPathMode,
  handleDetectVersion,
  handleAddCustomPath,
  handleRemoveCustomPath,
  startAddCustomPath,
  cancelAddCustomPath,
} = useCustomSdkPaths({
  notifyError,
  formatError: getErrorMessage,
  refreshNonVfoxSdks,
  checkCompatMode,
});

const sidebarAction = computed(() => props.sidebarAction);
useSdkManagerLifecycle({
  activeView,
  sidebarAction,
  loadPlatformInfo,
  closeDetail,
  emitSidebarActionDone: actionId => emit('sidebar-action-done', actionId),
});

const {
  removingPlugin,
  confirmAction,
  removePluginName,
  removePluginCustomSdks,
  requestUninstall,
  requestRemoveCustomPath,
  requestRemovePlugin,
  executeConfirm,
  executeRemovePlugin,
} = useSdkRemovalDialogs({
  runTask,
  notifyError,
  notifyTaskError,
  formatError: getErrorMessage,
  handleUninstall,
  removeCurrentCustomSdkBeforeDelete,
  handleRemoveCustomPath,
  fetchAllSdks,
  closeDetail,
});

const {
  copiedPath,
  copyPath,
} = useCopyablePath({
  notifyInfo,
  notifyError,
  formatError: getErrorMessage,
});

const {
  expandedVersions,
  toggleExpand,
} = useExpandableVersions();

const openPluginMarket = () => {
  emit('open-plugin-market');
};

const openSync = () => {
  emit('open-sync');
};
</script>

<template>
  <SdkManagerShell
    v-bind="{
      activeView, transitionName, loading, sdks, vfoxSdks, systemSdks, selectedSdk,
      checkingCompat, isPathOverrideApplied, hijacking, restoring, removingPlugin,
      pathOverrideTooltip, pathOverrideRemoveTooltip,
      hasDetail: Boolean(selectedSdk && sdkDetails[selectedSdk.name]),
      hasDetailError: Boolean(selectedSdk && detailError[selectedSdk.name]),
      unifiedVersions, expandedVersions, copiedPath, usingVersion, unusingSdk,
      searchingFor, searchLoading, filteredSearchResults, searchQuery,
      isAddingPathMode, customPathInput, customVersionInput, isAddingCustomPath,
      confirmAction, removePluginName, removePluginCustomSdks,
      getTotalVersionCount, displayVersion, truncateVersion, isSearchVersionInstalled,
    }"
    @open-detail="openDetail"
    @open-plugin-market="openPluginMarket"
    @open-sync="openSync"
    @back="closeDetail"
    @enable-path-override="handleHijackPlugin"
    @disable-path-override="handleRestorePlugin"
    @remove-plugin="requestRemovePlugin"
    @toggle-expand="toggleExpand"
    @copy-path="copyPath"
    @use-custom-path="handleUseCustomPath"
    @use-version="handleUse"
    @unuse="handleUnuse"
    @remove-custom-path="requestRemoveCustomPath"
    @uninstall-version="requestUninstall"
    @search="handleSearch"
    @install="handleInstall"
    @cancel-search="cancelSearch"
    @start-add-custom="startAddCustomPath"
    @cancel-add-custom="cancelAddCustomPath"
    @detect-version="handleDetectVersion"
    @add-custom-path="handleAddCustomPath"
    @update:search-query="searchQuery = $event"
    @update:custom-path-input="customPathInput = $event"
    @update:custom-version-input="customVersionInput = $event"
    @confirm-action="executeConfirm"
    @cancel-action="confirmAction.type = null"
    @confirm-remove-plugin="executeRemovePlugin"
    @cancel-remove-plugin="removePluginName = null"
  />
</template>
