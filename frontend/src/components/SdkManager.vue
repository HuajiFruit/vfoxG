<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { GetAllSdks, GetInstalledSdks, GetSdkDetail, UseVersion, UnuseVersion, InstallVersion, UninstallVersion, SearchVersions, GetVersionPath, RemovePluginWithOptions, GetNonVfoxSdks, AddNonVfoxSdk, RemoveNonVfoxSdk, ScanSystemSdks, UseCustomSdk, DetectSdkPathVersion, HijackPluginSystemPath, RestorePluginSystemPath, CheckPluginPathOverride, GetActiveCustomSdk, GetPlatformInfo, ExportCurrentEnvironmentSdks, ImportSdkEnvironmentFromTxt } from '../../wailsjs/go/main/App';
import { EventsOn, ClipboardSetText } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import PluginIcon from './PluginIcon.vue';
import ConfirmModal from './ConfirmModal.vue';
import RemovePluginModal from './RemovePluginModal.vue';
import { t } from '../i18n';

const sdks = ref<main.SdkInfo[]>([]);
const loading = ref(true);
const platformInfo = ref<main.PlatformInfo | null>(null);

const searchingFor = ref<string | null>(null);
const searchResults = ref<string[]>([]);
const searchLoading = ref(false);
const searchQuery = ref('');

const sdkDetails = ref<Record<string, main.SdkDetail>>({});
const detailError = ref<Record<string, boolean>>({});

// Master-Detail State
const activeView = ref<'list' | 'detail'>('list');
const transitionName = ref('fade-slide-forward');
const selectedSdk = ref<main.SdkInfo | null>(null);
const versionPaths = ref<Record<string, Record<string, string>>>({});
const usingVersion = ref<string | null>(null);
const exportingSdks = ref(false);
const importingSdks = ref(false);

const removingPlugin = ref<string | null>(null);

const emit = defineEmits(['start-task', 'notify']);
let sdksFetchSeq = 0;
let detailFetchSeq = 0;
let compatFetchSeq = 0;
let detailResetTimer: ReturnType<typeof setTimeout> | null = null;

const getErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err.trim()) return err;
  return fallback;
};

const notifyError = (message: string, title = t('common.error')) => {
  emit('notify', { type: 'error', title, message });
};

const notifySuccess = (message: string, title = t('common.success')) => {
  emit('notify', { type: 'success', title, message });
};

const notifyInfo = (message: string, title = t('common.notification')) => {
  emit('notify', { type: 'info', title, message });
};

const clearDetailResetTimer = () => {
  if (detailResetTimer !== null) {
    clearTimeout(detailResetTimer);
    detailResetTimer = null;
  }
};

const vfoxSdks = computed(() => sdks.value.filter(s => s.source === 'vfox'));
const systemSdks = computed(() => sdks.value.filter(s => s.source !== 'vfox'));
const filteredSearchResults = computed(() => {
  if (!searchQuery.value) return searchResults.value;
  const q = searchQuery.value.toLowerCase();
  return searchResults.value.filter(v => v.toLowerCase().includes(q));
});

const nonVfoxSdksMap = ref<Record<string, main.SdkInfo[]>>({});

const matchingNonVfoxSdks = computed(() => {
  if (!selectedSdk.value) return [];
  return nonVfoxSdksMap.value[selectedSdk.value.name] || [];
});

interface UnifiedVersion {
  isCustom: boolean;
  version: string;
  path: string;
  isCurrent: boolean;
  sysSdk?: main.SdkInfo;
  vfoxVersion?: string;
}

const unifiedVersions = computed<UnifiedVersion[]>(() => {
  if (!selectedSdk.value) return [];
  const versions: UnifiedVersion[] = [];
  
  // Add Vfox versions
  const detail = sdkDetails.value[selectedSdk.value.name];
  if (detail && detail.versions) {
    for (const v of detail.versions) {
      versions.push({
        isCustom: false,
        version: v.version,
        path: versionPaths.value[selectedSdk.value.name]?.[v.version] || '',
        isCurrent: v.isCurrent,
        vfoxVersion: v.version
      });
    }
  }

  // Add Custom versions
  for (const sysSdk of matchingNonVfoxSdks.value) {
    versions.push({
      isCustom: true,
      version: sysSdk.versions?.[0]?.version || 'unknown',
      path: sysSdk.path,
      isCurrent: (activeCustomSdk.value || '').toLowerCase() === (sysSdk.path || '').toLowerCase(),
      sysSdk: sysSdk
    });
  }

  return versions;
});

const customPathInput = ref('');
const customVersionInput = ref('');
const isDetectingVersion = ref(false);
const isAddingCustomPath = ref(false);
const isAddingPathMode = ref<string | null>(null);

const handleDetectVersion = async (name: string) => {
  if (!customPathInput.value.trim()) return;
  isDetectingVersion.value = true;
  try {
    const v = await DetectSdkPathVersion(name, customPathInput.value.trim());
    if (v && v !== 'unknown') {
      customVersionInput.value = v;
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.detect_error')));
  } finally {
    isDetectingVersion.value = false;
  }
};

const handleAddCustomPath = async (name: string) => {
  if (!customPathInput.value.trim()) return;
  isAddingCustomPath.value = true;
  try {
    await AddNonVfoxSdk(name, customPathInput.value.trim(), customVersionInput.value.trim());
    customPathInput.value = '';
    customVersionInput.value = '';
    isAddingPathMode.value = null;
    nonVfoxSdksMap.value = await GetNonVfoxSdks();
    await checkCompatMode(name);
  } catch (err: any) {
    notifyError(getErrorMessage(err, t('sdk.custom.add_error')));
  } finally {
    isAddingCustomPath.value = false;
  }
};

const handleRemoveCustomPath = async (name: string, path: string) => {
  try {
    await RemoveNonVfoxSdk(name, path);
    nonVfoxSdksMap.value = await GetNonVfoxSdks();
    await checkCompatMode(name);
  } catch (err: any) {
    notifyError(getErrorMessage(err, t('sdk.custom.remove_error')));
  }
};

const handleUseCustomPath = async (name: string, path: string) => {
  emit('start-task', t('task.custom.use', { name, path }));
  usingVersion.value = path;
  try {
    const result = await UseCustomSdk(name, path);
    if (result !== 'ok') {
      notifyError(t('sdk.custom.apply_error'));
    }
    await checkCompatMode(name);
  } catch (e) {
    notifyError(getErrorMessage(e, t('sdk.custom.apply_exception')));
  } finally {
    usingVersion.value = null;
  }
};

const getTotalVersionCount = (sdk: main.SdkInfo) => {
  const vfoxCount = sdk.versions?.length || 0;
  const systemCount = nonVfoxSdksMap.value[sdk.name]?.length || 0;
  return vfoxCount + systemCount;
};

const formatVersionTitle = (name: string, rawVersion: string) => {
  if (!rawVersion) return 'unknown';
  // Remove the SDK name (e.g. "Python ", "Node ") case-insensitively from the start
  const regex = new RegExp(`^${name}\\s*`, 'i');
  return rawVersion.replace(regex, '').trim();
};

const displayVersion = (version?: string) => version || t('common.unknown');

const truncateVersion = (version?: string, maxLength = 30) => {
  const text = displayVersion(version);
  return text.length > maxLength ? `${text.substring(0, maxLength)}...` : text;
};

const mergeSdkLists = (vfox: main.SdkInfo[], system: main.SdkInfo[]) => {
  const vfoxNames = new Set(vfox.map(s => s.name));
  return [...vfox, ...system.filter(s => !vfoxNames.has(s.name))];
};

const fetchVfoxSdks = async () => {
  try {
    const vfox = await GetInstalledSdks();
    const system = sdks.value.filter(s => s.source !== 'vfox');
    sdks.value = mergeSdkLists(vfox, system);
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.refresh_error')));
  }
};

const fetchAllSdks = async () => {
  const requestId = ++sdksFetchSeq;
  loading.value = true;
  try {
    const all = await GetAllSdks();
    if (requestId !== sdksFetchSeq) return;
    sdks.value = all;
  } catch (err) {
    if (requestId === sdksFetchSeq) {
      notifyError(getErrorMessage(err, t('sdk.load_error')));
    }
  } finally {
    if (requestId === sdksFetchSeq) {
      loading.value = false;
    }
  }
};

const handleExportSdks = async () => {
  exportingSdks.value = true;
  try {
    const path = await ExportCurrentEnvironmentSdks();
    if (path) {
      notifySuccess(t('sdk.export.success', { path }));
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.export.error')));
  } finally {
    exportingSdks.value = false;
  }
};

const handleImportSdks = async () => {
  importingSdks.value = true;
  try {
    const result = await ImportSdkEnvironmentFromTxt();
    if (result?.path) {
      await fetchAllSdks();
      nonVfoxSdksMap.value = await GetNonVfoxSdks();
      const message = t('sdk.import.success', {
        imported: result.importedCustomSdks ?? 0,
        skipped: result.skippedCustomSdks ?? 0,
        vfox: result.vfoxSdksFound ?? 0,
      });
      const warningText = result.warnings?.length ? ` ${result.warnings.join(' ')}` : '';
      notifySuccess(`${message}${warningText}`);
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.import.error')));
  } finally {
    importingSdks.value = false;
  }
};

let systemReadyOff: (() => void) | null = null;

onMounted(async () => {
  await loadPlatformInfo();
  await fetchAllSdks();
  let mounted = true;

  const offSystem = EventsOn('system-sdks-ready', () => {
    if (!mounted) return;
    GetAllSdks().then(all => { if (mounted) sdks.value = all; }).catch(() => {});
  });

  const offSdkChanged = EventsOn('sdk-list-changed', () => {
    if (!mounted) return;
    fetchAllSdks();
    GetNonVfoxSdks().then(res => { if (mounted) nonVfoxSdksMap.value = res; }).catch(() => {});
    if (selectedSdk.value) {
      fetchDetail(selectedSdk.value.name);
      checkCompatMode(selectedSdk.value.name);
    }
  });

  GetNonVfoxSdks().then(res => {
    if (mounted) nonVfoxSdksMap.value = res;
  }).catch(() => {});

  systemReadyOff = () => { mounted = false; offSystem(); offSdkChanged(); };
});

onUnmounted(() => {
  if (systemReadyOff) systemReadyOff();
});

const fetchDetail = async (name: string, requestId = ++detailFetchSeq) => {
  detailError.value[name] = false;
  try {
    const detail = await GetSdkDetail(name);
    if (requestId !== detailFetchSeq) return;
    sdkDetails.value[name] = detail;
  } catch (err) {
    if (requestId === detailFetchSeq) {
      detailError.value[name] = true;
      notifyError(getErrorMessage(err, t('sdk.detail_error', { name })));
    }
  }
};

const openDetail = async (sdk: main.SdkInfo) => {
  clearDetailResetTimer();
  const requestId = ++detailFetchSeq;
  selectedSdk.value = sdk;
  transitionName.value = 'fade-slide-forward';
  activeView.value = 'detail';
  versionPaths.value[sdk.name] = {};

  // 统一尝试拉取 vfox 详情（有 plugin 安装的才能拉到）
  await fetchDetail(sdk.name, requestId);
  if (requestId !== detailFetchSeq || selectedSdk.value?.name !== sdk.name) return;
  if (sdkDetails.value[sdk.name]?.versions) {
    for (const v of sdkDetails.value[sdk.name].versions) {
      try {
        const path = await GetVersionPath(sdk.name, v.version);
        if (requestId !== detailFetchSeq || selectedSdk.value?.name !== sdk.name) return;
        versionPaths.value[sdk.name][v.version] = path;
      } catch(e) {
        if (requestId === detailFetchSeq) {
          notifyError(getErrorMessage(e, t('sdk.path.load_error', { name: sdk.name, version: v.version })));
        }
      }
    }
  }

  // 拉取 non-vfox custom paths
  try {
    nonVfoxSdksMap.value = await GetNonVfoxSdks();
  } catch(e) {
    notifyError(getErrorMessage(e, t('sdk.custom_paths.load_error')));
  }
  
  await checkCompatMode(sdk.name);
};

const checkingCompat = ref(false);
const isPathOverrideApplied = ref(false);
const activeCustomSdk = ref<string>('');
const hijacking = ref(false);
const restoring = ref(false);

const pathOverrideTarget = computed(() => platformInfo.value?.sdkPathTarget || t('settings.system.path'));
const pathOverrideAdminText = computed(() => platformInfo.value?.requiresElevation ? t('platform.admin.required') : '');
const pathOverrideRestartHint = computed(() => {
  const os = platformInfo.value?.os || 'default';
  const key = `platform.restart.${os}`;
  const value = t(key);
  return value === key ? t('platform.restart.default') : value;
});
const pathOverrideTooltip = computed(() => t('sdk.path_override.tooltip', {
  target: pathOverrideTarget.value,
  restart: pathOverrideRestartHint.value,
  admin: pathOverrideAdminText.value,
}));
const pathOverrideRemoveTooltip = computed(() => t('sdk.path_override.remove_tooltip', {
  target: pathOverrideTarget.value,
  restart: pathOverrideRestartHint.value,
  admin: pathOverrideAdminText.value,
}));

const loadPlatformInfo = async () => {
  try {
    platformInfo.value = await GetPlatformInfo();
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.platform.load_error')));
  }
};

const checkCompatMode = async (name: string) => {
  const requestId = ++compatFetchSeq;
  checkingCompat.value = true;
  try {
    const applied = await CheckPluginPathOverride(name);
    const activeSdk = await GetActiveCustomSdk(name);
    if (requestId !== compatFetchSeq || selectedSdk.value?.name !== name) return;
    isPathOverrideApplied.value = applied;
    activeCustomSdk.value = activeSdk;
  } catch (err) {
    if (requestId === compatFetchSeq && selectedSdk.value?.name === name) {
      notifyError(getErrorMessage(err, t('sdk.path_override.check_error', { name })));
    }
  } finally {
    if (requestId === compatFetchSeq) {
      checkingCompat.value = false;
    }
  }
};

const handleHijackPlugin = async (name: string) => {
  hijacking.value = true;
  try {
    await HijackPluginSystemPath(name);
    await checkCompatMode(name);
    notifySuccess(t('sdk.path_override.enable_success', { name, restart: pathOverrideRestartHint.value }));
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.path_override.enable_error', { name })));
  } finally {
    hijacking.value = false;
  }
};

const handleRestorePlugin = async (name: string) => {
  restoring.value = true;
  try {
    await RestorePluginSystemPath(name);
    await checkCompatMode(name);
    notifySuccess(t('sdk.path_override.disable_success', { name, restart: pathOverrideRestartHint.value }));
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.path_override.disable_error', { name })));
  } finally {
    restoring.value = false;
  }
};

const closeDetail = () => {
  clearDetailResetTimer();
  ++detailFetchSeq;
  ++compatFetchSeq;
  transitionName.value = 'fade-slide-backward';
  activeView.value = 'list';
  detailResetTimer = setTimeout(() => {
    selectedSdk.value = null;
    searchingFor.value = null;
    searchResults.value = [];
    searchQuery.value = '';
    detailResetTimer = null;
  }, 300);
};

const handleUse = async (name: string, version: string) => {
  emit('start-task', t('task.version.switch', { name, version }));
  usingVersion.value = version;
  try {
    await UseVersion(name, version);
    // UseVersion 异步执行，由 sdk-list-changed 事件触发刷新
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.switch_error', { name, version })));
  } finally {
    usingVersion.value = null;
  }
};

const handleUnuse = async (name: string) => {
  emit('start-task', t('task.version.unset', { name }));
  try {
    await UnuseVersion(name);
    // UnuseVersion 异步执行，由 sdk-list-changed 事件触发刷新
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.unset_error', { name })));
  }
};

const handleInstall = async (name: string, version: string) => {
  emit('start-task', t('task.version.install', { name, version }));
  try {
    await InstallVersion(name, version);
    await fetchDetail(name, ++detailFetchSeq);
    await fetchVfoxSdks();
    
    // Fetch path for newly installed version
    const newPath = await GetVersionPath(name, version);
    versionPaths.value[name] ||= {};
    versionPaths.value[name][version] = newPath;
    
    if (searchingFor.value === name) {
      searchResults.value = [];
      searchingFor.value = null;
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.install_error', { name, version })));
  }
};


const confirmAction = ref<{ type: 'removePlugin' | 'uninstallVersion' | 'removeCustomSdk' | null; name: string; version?: string; path?: string }>({ type: null, name: '' });

const requestUninstall = (name: string, version: string) => {
  confirmAction.value = { type: 'uninstallVersion', name, version };
};

const requestRemovePlugin = async (name: string) => {
  // Fetch custom SDKs for this plugin to show the choice modal
  try {
    const nonVfoxMap = await GetNonVfoxSdks();
    const sdks = nonVfoxMap[name] || [];
    removePluginCustomSdks.value = sdks.map((s: any) => ({
      path: s.path || s.Path || '',
      version: s.versions?.[0]?.version || s.version || s.Version || 'unknown',
    }));
  } catch (err) {
    removePluginCustomSdks.value = [];
    notifyError(getErrorMessage(err, t('market.custom_refs_error', { name })));
  }
  removePluginName.value = name;
};

const requestRemoveCustomPath = (name: string, path: string) => {
  confirmAction.value = { type: 'removeCustomSdk', name, path };
};

const executeConfirm = async () => {
  const { type, name, version, path } = confirmAction.value;
  confirmAction.value = { type: null, name: '' };
  
  if (type === 'uninstallVersion' && version) {
    emit('start-task', t('task.version.uninstall', { name, version }));
    try {
      if (sdkDetails.value[name]?.current === version) {
        await handleUnuse(name);
      }
      await UninstallVersion(name, version);
      await fetchDetail(name, ++detailFetchSeq);
      await fetchVfoxSdks();
      const paths = versionPaths.value[name];
      if (paths) {
        delete paths[version];
        if (Object.keys(paths).length === 0) {
          delete versionPaths.value[name];
        }
      }
    } catch (err) {
      notifyError(getErrorMessage(err, t('sdk.uninstall_error', { name, version })));
    }
  } else if (type === 'removeCustomSdk' && path) {
    const isCurrent = activeCustomSdk.value === path;
    if (isCurrent) {
      await handleUnuse(name);
    }
    await handleRemoveCustomPath(name, path);
  }
};

// RemovePluginModal state
const removePluginName = ref<string | null>(null);
const removePluginCustomSdks = ref<Array<{ path: string; version: string }>>([]);

const executeRemovePlugin = async (chosenPath: string | null) => {
  const name = removePluginName.value;
  removePluginName.value = null;
  if (!name) return;

  emit('start-task', t('task.plugin.remove', { name }));
  removingPlugin.value = name;
  try {
    await RemovePluginWithOptions(name, chosenPath || '');
    await fetchAllSdks();
    closeDetail();
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.remove_plugin_error', { name })));
  } finally {
    removingPlugin.value = null;
  }
};

const handleSearch = async (name: string) => {
  searchingFor.value = name;
  searchQuery.value = '';
  searchLoading.value = true;
  try {
    const results = await SearchVersions(name);
    if (searchingFor.value === name) {
      searchResults.value = results;
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.search_error', { name })));
    if (searchingFor.value === name) {
      searchResults.value = [];
    }
  } finally {
    if (searchingFor.value === name) {
      searchLoading.value = false;
    }
  }
};

const copiedPath = ref<string | null>(null);

const copyPath = async (path: string) => {
  if (path) {
    try {
      await ClipboardSetText(path);
      copiedPath.value = path;
      notifyInfo(t('sdk.path.copied'));
      setTimeout(() => {
        if (copiedPath.value === path) {
          copiedPath.value = null;
        }
      }, 2000);
    } catch (err) {
      notifyError(getErrorMessage(err, t('sdk.path.copy_error')));
    }
  }
};

const expandedVersions = ref<Record<string, boolean>>({});
const toggleExpand = (id: string) => {
  expandedVersions.value[id] = !expandedVersions.value[id];
};
</script>

<template>
  <div class="sdk-manager">
    <!-- MAIN VIEW (LIST) -->
    <Transition :name="transitionName" mode="out-in">
      <div v-if="activeView === 'list'" key="list" class="view-container">
        <div class="sdk-list-header">
          <h2>{{ t('sdk.installed.title') }}</h2>
          <div class="sdk-list-actions">
            <button class="btn tonal small" :disabled="importingSdks || exportingSdks" @click="handleImportSdks">
              <div v-if="importingSdks" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px;"></div>
              <template v-else>
                <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">upload_file</span>
                {{ t('sdk.import') }}
              </template>
            </button>
            <button class="btn tonal small" :disabled="exportingSdks || importingSdks" @click="handleExportSdks">
              <div v-if="exportingSdks" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px;"></div>
              <template v-else>
                <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">download</span>
                {{ t('sdk.export') }}
              </template>
            </button>
          </div>
        </div>
        
        <div v-if="loading" class="flex-center" style="height: 200px;">
          <div class="spinner"></div>
        </div>

        <div v-else-if="sdks.length === 0" class="empty-state" style="text-align: center; padding: 40px;">
          {{ t('sdk.installed.empty') }}
        </div>

        <template v-else>
          <template v-if="vfoxSdks.length">
            <h3 class="section-heading">{{ t('sdk.vfox.title') }}</h3>
            <div class="sdk-grid">
              <div v-for="sdk in vfoxSdks" :key="sdk.name" class="sdk-card" @click="openDetail(sdk)">
                <PluginIcon :name="sdk.name" class="sdk-icon-large" />
                <div class="sdk-card-content">
                  <h3>{{ sdk.name }}</h3>
                  <span class="version-count">
                    {{ getTotalVersionCount(sdk) }} 
                    {{ getTotalVersionCount(sdk) !== 1 ? t('sdk.versions') : t('sdk.version') }}
                  </span>
                </div>
              </div>
            </div>
          </template>

          <template v-if="systemSdks.length">
            <h3 class="section-heading" style="margin-top: 32px;">{{ t('sdk.nonevfox.title') }}</h3>
            <div class="sdk-grid">
              <div v-for="sdk in systemSdks" :key="sdk.name" class="sdk-card card-system" @click="openDetail(sdk)">
                <PluginIcon :name="sdk.name" class="sdk-icon-large" />
                <div class="sdk-card-content">
                  <h3>{{ sdk.name }}</h3>
                  <span class="version-count" :title="displayVersion(sdk.versions?.[0]?.version)">{{ truncateVersion(sdk.versions?.[0]?.version) }}</span>
                </div>
              </div>
            </div>
          </template>
        </template>
      </div>

      <!-- SECONDARY PAGE (DETAIL) -->
      <div v-else-if="activeView === 'detail' && selectedSdk" key="detail" class="view-container detail-view">
        <div class="detail-header">
          <button class="btn tonal small back-btn" @click="closeDetail">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
            {{ t('sdk.back') }}
          </button>
          
          <div class="detail-title-area" style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
            <div style="display: flex; align-items: center; gap: 24px;">
              <PluginIcon :name="selectedSdk.name" class="sdk-icon-huge" />
              <div class="detail-title-text">
                <h1>{{ selectedSdk.name }}</h1>
              </div>
            </div>
            
            <div v-if="selectedSdk.source === 'vfox'" style="display: flex; gap: 8px; align-items: flex-start;">
              <div style="display: flex; flex-direction: column; align-items: center; gap: 4px;">
                <button 
                  v-if="checkingCompat"
                  class="btn tonal small" 
                  disabled
                  style="min-width: 140px; display: flex; justify-content: center; align-items: center; background: transparent;"
                >
                  <div class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px; border-color: var(--md-outline) transparent var(--md-outline) transparent;"></div>
                </button>
                <button 
                  v-else-if="!isPathOverrideApplied"
                  class="btn tonal small" 
                  @click="handleHijackPlugin(selectedSdk.name)" 
                  :disabled="hijacking || restoring" 
                  style="min-width: 140px; display: flex; justify-content: center; align-items: center;"
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
                  @click="handleRestorePlugin(selectedSdk.name)" 
                  :disabled="hijacking || restoring" 
                  style="min-width: 140px; display: flex; justify-content: center; align-items: center;"
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
                <span v-if="!checkingCompat && !isPathOverrideApplied" style="font-size: 11px; color: var(--md-outline); opacity: 0.8;">{{ t('sdk.path_override.hint') }}</span>
              </div>

              <button 
                class="btn tonal small" 
                style="color: var(--md-error); background: rgba(239, 68, 68, 0.1); min-width: 120px; display: flex; justify-content: center; align-items: center;" 
                :disabled="removingPlugin === selectedSdk.name"
                @click="requestRemovePlugin(selectedSdk.name)"
              >
                <div v-if="removingPlugin === selectedSdk.name" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px; border-top-color: var(--md-error);"></div>
                <template v-else>
                  <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">delete</span>
                  {{ t('sdk.remove_plugin') }}
                </template>
              </button>
            </div>
          </div>
        </div>

        <div class="detail-body">
          <!-- Unified detail: show whatever data is available -->
          <!-- Unified detail: show whatever data is available -->
          
          <!-- UNMANAGED SYSTEM SDK VIEW -->
          <div v-if="selectedSdk.source === 'system'">
            <div class="vfox-versions-section">
              <h2>{{ t('sdk.nonevfox.title') }}</h2>
              <div class="version-card">
                <div class="version-card-header">
                  <div class="flex-align-center flex-gap-12" style="min-width: 0; flex: 1;">
                    <h3 class="version-title" :style="{ display: 'flex', alignItems: expandedVersions['sys-' + selectedSdk.path] ? 'flex-start' : 'center', flexWrap: expandedVersions['sys-' + selectedSdk.path] ? 'wrap' : 'nowrap', gap: '8px', minWidth: 0, flex: '0 1 auto' }">
                      <span :style="{ wordBreak: 'break-all', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: expandedVersions['sys-' + selectedSdk.path] ? 'normal' : 'nowrap' }">
                        {{ displayVersion(selectedSdk.versions?.[0]?.version) }}
                      </span>
                      <button v-if="displayVersion(selectedSdk.versions?.[0]?.version).length > 50" class="btn text small" style="padding: 0 4px; min-width: auto; height: 20px; line-height: 20px; flex-shrink: 0;" @click="toggleExpand('sys-' + selectedSdk.path)">
                        {{ expandedVersions['sys-' + selectedSdk.path] ? t('common.collapse') : t('common.expand') }}
                      </button>
                    </h3>
                    <span class="system-tag" style="flex-shrink: 0;">{{ t('sdk.custom') }}</span>
                  </div>
                </div>
                <div class="version-card-body">
                  <div class="path-label">{{ t('sdk.exe_path') }}</div>
                  <div class="path-box">
                    <code class="path-text">{{ selectedSdk.path }}</code>
                    <button class="btn icon-btn" @click="copyPath(selectedSdk.path)" :title="t('common.copy_path')">
                      <svg v-if="copiedPath === selectedSdk.path" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
                      <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                    </button>
                  </div>
                  <p class="empty-state" style="margin-top: 12px; font-size: 14px; text-align: left;">
                    {{ t('sdk.system.manage_hint') }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- MANAGED SDK VIEW (Unified) -->
          <div v-else>
            <div class="vfox-versions-section">
              <h2>{{ t('sdk.version_list') }}</h2>
              
              <div v-if="!sdkDetails[selectedSdk.name] && !detailError[selectedSdk.name]" class="flex-center" style="padding: 20px;">
                <div class="spinner"></div>
              </div>

              <div v-else-if="unifiedVersions.length" class="versions-grid">
                <div v-for="ver in unifiedVersions" :key="ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version" class="version-card" :class="{ 'is-current': ver.isCurrent }">
                  <div class="version-card-header">
                    <div class="flex-align-center flex-gap-12" style="min-width: 0; flex: 1;">
                      <h3 class="version-title" :style="{ display: 'flex', alignItems: expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? 'flex-start' : 'center', flexWrap: expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? 'wrap' : 'nowrap', gap: '8px', minWidth: 0, flex: '0 1 auto' }">
                        <span :style="{ wordBreak: 'break-all', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? 'normal' : 'nowrap' }">
                          {{ ver.version }}
                        </span>
                        <button v-if="ver.version.length > 50" class="btn text small" style="padding: 0 4px; min-width: auto; height: 20px; line-height: 20px; flex-shrink: 0;" @click="toggleExpand(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)">
                          {{ expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? t('common.collapse') : t('common.expand') }}
                        </button>
                      </h3>
                      <span v-if="ver.isCurrent" class="current-tag" style="flex-shrink: 0;">{{ t('sdk.current') }}</span>
                      <span v-if="ver.isCustom" class="system-tag" style="flex-shrink: 0;">{{ t('sdk.custom') }}</span>
                      <span v-else class="vfox-tag" style="flex-shrink: 0;">vfox</span>
                    </div>
                    <div class="version-actions">
                      <button v-if="!ver.isCurrent" class="btn tonal small" :disabled="usingVersion === (ver.isCustom ? ver.path : ver.version)" @click="ver.isCustom ? handleUseCustomPath(selectedSdk.name, ver.path) : handleUse(selectedSdk.name, ver.vfoxVersion!)">{{ usingVersion === (ver.isCustom ? ver.path : ver.version) ? '...' : t('sdk.use') }}</button>
                      <button v-if="ver.isCurrent" class="btn text small danger" @click="handleUnuse(selectedSdk.name)">{{ t('sdk.unset') }}</button>
                      <button v-if="ver.isCustom" class="btn text small danger" @click="requestRemoveCustomPath(selectedSdk.name, ver.path)">{{ t('sdk.remove') }}</button>
                      <button v-else class="btn text small danger" @click="requestUninstall(selectedSdk.name, ver.vfoxVersion!)">{{ t('sdk.uninstall') }}</button>
                    </div>
                  </div>
                  <div class="version-card-body">
                    <div class="path-label">{{ ver.isCustom ? t('sdk.exe_path') : t('sdk.install_path') }}</div>
                    <div class="path-box" v-if="ver.path">
                      <code class="path-text">{{ ver.path }}</code>
                      <button class="btn icon-btn" @click="copyPath(ver.path)" :title="t('common.copy_path')">
                        <svg v-if="copiedPath === ver.path" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
                        <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                      </button>
                    </div>
                    <div v-else class="path-box loading-path">
                      <div class="spinner small"></div> {{ t('sdk.loading_path') }}
                    </div>
                  </div>
                </div>
              </div>

              <div v-else class="empty-state">{{ t('sdk.no_versions_installed') }}</div>

              <!-- Install Section (vfox only) -->
              <div v-if="selectedSdk.source === 'vfox'" class="install-section-large" style="margin-top: 24px; padding-top: 24px; border-top: 1px dashed var(--panel-border, var(--md-outline-variant));">
                <button v-if="searchingFor !== selectedSdk.name" class="btn primary large-btn" @click="handleSearch(selectedSdk.name)">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                  {{ t('sdk.install_new') }}
                </button>

                <div v-else class="search-box large">
                  <div class="search-header">
                    <input v-model="searchQuery" type="text" class="search-input" :placeholder="t('sdk.search_versions.placeholder')" autofocus />
                    <button class="btn text" @click="searchingFor = null">{{ t('common.cancel') }}</button>
                  </div>
                  <div v-if="searchLoading" class="flex-center" style="padding: 24px;"><div class="spinner"></div></div>
                  <div v-else class="search-results-grid">
                    <button v-for="ver in filteredSearchResults" :key="ver" class="search-result-card" @click="handleInstall(selectedSdk.name, ver)">
                      {{ ver }}
                      <span class="install-text">{{ t('market.install') }}</span>
                    </button>
                    <div v-if="!filteredSearchResults.length" class="empty-state" style="grid-column: 1/-1;">{{ t('sdk.no_matching_versions') }}</div>
                  </div>
                </div>
              </div>

              <!-- Add Custom Path Form -->
              <div class="install-section-large" style="margin-top: 24px; padding-top: 24px; border-top: 1px dashed var(--panel-border, var(--md-outline-variant));">
                <button v-if="isAddingPathMode !== selectedSdk.name" class="btn tonal small" @click="isAddingPathMode = selectedSdk.name">+ {{ t('sdk.add_custom') }}</button>
                <div v-else class="search-box large">
                  <div class="search-header">
                    <input v-model="customPathInput" @blur="handleDetectVersion(selectedSdk.name)" type="text" class="search-input" :placeholder="t('sdk.custom_path.placeholder')" style="font-size: 14px; padding: 12px 16px; flex: 2; box-sizing: border-box; margin: 0; height: 100%; min-height: 44px;" autofocus />
                    <input v-model="customVersionInput" type="text" class="search-input" :placeholder="t('sdk.version.placeholder')" style="font-size: 14px; padding: 12px 16px; flex: 1; box-sizing: border-box; margin: 0; height: 100%;" />
                    <button class="btn text" @click="isAddingPathMode = null; customPathInput = ''; customVersionInput = ''">{{ t('sdk.cancel') }}</button>
                    <button class="btn primary" :disabled="isAddingCustomPath || !customPathInput.trim()" @click="handleAddCustomPath(selectedSdk.name)" style="min-width: 80px; display: flex; justify-content: center;">
                      <div v-if="isAddingCustomPath" class="spinner small-spinner" style="width: 14px; height: 14px; border-width: 2px;"></div>
                      <template v-else>{{ t('sdk.add') }}</template>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>

    <Teleport to="body">
      <ConfirmModal 
        v-if="confirmAction.type"
        :title="confirmAction.type === 'uninstallVersion' ? t('sdk.uninstall') : t('sdk.remove')"
        :message="confirmAction.type === 'uninstallVersion'
          ? t('sdk.confirm.uninstall_version_message', { name: confirmAction.name, version: confirmAction.version || '' })
          : t('sdk.confirm.remove_custom_message', { note: t('sdk.confirm.note'), question: t('sdk.confirm.remove_custom') })"
        :danger="true"
        :confirmText="confirmAction.type === 'uninstallVersion' ? t('sdk.uninstall') : t('sdk.remove_reference')"
        @confirm="executeConfirm"
        @cancel="confirmAction.type = null"
      />
      <RemovePluginModal
        v-if="removePluginName"
        :pluginName="removePluginName"
        :customSdks="removePluginCustomSdks"
        @confirm="executeRemovePlugin"
        @cancel="removePluginName = null"
      />
    </Teleport>
  </div>
</template>
