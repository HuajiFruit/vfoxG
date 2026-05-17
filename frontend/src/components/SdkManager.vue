<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { GetAllSdks, GetInstalledSdks, GetSdkDetail, UseVersion, UnuseVersion, InstallVersion, UninstallVersion, SearchVersions, GetVersionPath, RemovePlugin, GetNonVfoxSdks, AddNonVfoxSdk, RemoveNonVfoxSdk, ScanSystemSdks, UseCustomSdk, DetectSdkPathVersion, RestoreSystemPath, HijackPluginSystemPath, RestorePluginSystemPath, CheckPluginWin11CompatMode, GetActiveCustomSdk } from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff, ClipboardSetText } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import PluginIcon from './PluginIcon.vue';
import ConfirmModal from './ConfirmModal.vue';
import { t } from '../i18n';

const sdks = ref<main.SdkInfo[]>([]);
const loading = ref(true);
const errorMsg = ref('');

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
const versionPaths = ref<Record<string, string>>({});
const usingVersion = ref<string | null>(null);

const removingPlugin = ref<string | null>(null);

const emit = defineEmits(['start-task']);

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
        path: versionPaths.value[v.version] || '',
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
    console.error('Failed to detect version:', err);
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
    errorMsg.value = err.message || 'Failed to add custom path';
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
    errorMsg.value = err.message || 'Failed to remove custom path';
  }
};

const handleUseCustomPath = async (name: string, path: string) => {
  emit('start-task', `Using ${name} (${path})`);
  usingVersion.value = path;
  try {
    activeCustomSdk.value = path;
    const detail = sdkDetails.value[name];
    if (detail) {
      detail.current = '';
      detail.versions.forEach(v => { v.isCurrent = false; });
    }
    const result = await UseCustomSdk(name, path);
    if (result !== 'ok') {
      errorMsg.value = 'Failed to apply custom SDK.';
    }
    await checkCompatMode(name);
  } catch (e) {
    console.error(e);
    errorMsg.value = 'Error applying custom SDK.';
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
    console.error(err);
  }
};

const fetchAllSdks = async () => {
  loading.value = true;
  errorMsg.value = '';
  try {
    sdks.value = await GetAllSdks();
  } catch (err) {
    console.error(err);
    errorMsg.value = 'Failed to load SDK list.';
  } finally {
    loading.value = false;
  }
};

let systemReadyOff: (() => void) | null = null;

onMounted(async () => {
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

const fetchDetail = async (name: string) => {
  detailError.value[name] = false;
  try {
    const detail = await GetSdkDetail(name);
    sdkDetails.value[name] = detail;
  } catch (err) {
    console.error(err);
    detailError.value[name] = true;
  }
};

const openDetail = async (sdk: main.SdkInfo) => {
  selectedSdk.value = sdk;
  transitionName.value = 'fade-slide-forward';
  activeView.value = 'detail';
  versionPaths.value = {};

  // 统一尝试拉取 vfox 详情（有 plugin 安装的才能拉到）
  await fetchDetail(sdk.name);
  if (sdkDetails.value[sdk.name]?.versions) {
    sdkDetails.value[sdk.name].versions.forEach(async (v) => {
      try {
        const path = await GetVersionPath(sdk.name, v.version);
        versionPaths.value[v.version] = path;
      } catch(e) {}
    });
  }

  // 拉取 non-vfox custom paths
  try {
    nonVfoxSdksMap.value = await GetNonVfoxSdks();
  } catch(e) {}
  
  await checkCompatMode(sdk.name);
};

const checkingCompat = ref(false);
const isWin11CompatApplied = ref(false);
const activeCustomSdk = ref<string>('');
const hijacking = ref(false);
const restoring = ref(false);

const checkCompatMode = async (name: string) => {
  checkingCompat.value = true;
  try {
    isWin11CompatApplied.value = await CheckPluginWin11CompatMode(name);
    activeCustomSdk.value = await GetActiveCustomSdk(name);
  } catch (err) {
    console.error(err);
  } finally {
    checkingCompat.value = false;
  }
};

const handleHijackPlugin = async (name: string) => {
  hijacking.value = true;
  try {
    await HijackPluginSystemPath(name);
    await checkCompatMode(name);
  } catch (err) {
    console.error(err);
  } finally {
    hijacking.value = false;
  }
};

const handleRestorePlugin = async (name: string) => {
  restoring.value = true;
  try {
    await RestorePluginSystemPath(name);
    await checkCompatMode(name);
  } catch (err) {
    console.error(err);
  } finally {
    restoring.value = false;
  }
};

const closeDetail = () => {
  transitionName.value = 'fade-slide-backward';
  activeView.value = 'list';
  setTimeout(() => {
    selectedSdk.value = null;
    searchingFor.value = null;
    searchResults.value = [];
    searchQuery.value = '';
  }, 300);
};

const handleUse = async (name: string, version: string) => {
  emit('start-task', `Switching ${name} to ${version}`);
  usingVersion.value = version;
  try {
    // 乐观更新：立即标记当前版本
    const detail = sdkDetails.value[name];
    if (detail) {
      detail.current = version;
      detail.versions.forEach(v => { v.isCurrent = v.version === version; });
    }
    activeCustomSdk.value = '';
    await UseVersion(name, version);
    // UseVersion 异步执行，由 sdk-list-changed 事件触发刷新
  } catch (err) {
    console.error(err);
    errorMsg.value = `Failed to switch ${name} to ${version}.`;
  } finally {
    usingVersion.value = null;
  }
};

const handleUnuse = async (name: string) => {
  emit('start-task', `Unsetting ${name}`);
  try {
    // 乐观更新：立即清除 current 状态
    const detail = sdkDetails.value[name];
    if (detail) {
      detail.current = '';
      detail.versions.forEach(v => { v.isCurrent = false; });
    }
    activeCustomSdk.value = '';
    await UnuseVersion(name);
    // UnuseVersion 异步执行，由 sdk-list-changed 事件触发刷新
  } catch (err) {
    console.error(err);
    errorMsg.value = `Failed to unset ${name}.`;
  }
};

const handleInstall = async (name: string, version: string) => {
  emit('start-task', `Installing ${name}@${version}`);
  try {
    await InstallVersion(name, version);
    await fetchDetail(name);
    await fetchVfoxSdks();
    
    // Fetch path for newly installed version
    const newPath = await GetVersionPath(name, version);
    versionPaths.value[version] = newPath;
    
    if (searchingFor.value === name) {
      searchResults.value = [];
      searchingFor.value = null;
    }
  } catch (err) {
    console.error(err);
    errorMsg.value = `Failed to install ${name}@${version}.`;
  }
};


const confirmAction = ref<{ type: 'removePlugin' | 'uninstallVersion' | 'removeCustomSdk' | null; name: string; version?: string; path?: string }>({ type: null, name: '' });

const requestUninstall = (name: string, version: string) => {
  confirmAction.value = { type: 'uninstallVersion', name, version };
};

const requestRemovePlugin = (name: string) => {
  confirmAction.value = { type: 'removePlugin', name };
};

const requestRemoveCustomPath = (name: string, path: string) => {
  confirmAction.value = { type: 'removeCustomSdk', name, path };
};

const executeConfirm = async () => {
  const { type, name, version, path } = confirmAction.value;
  confirmAction.value = { type: null, name: '' };
  
  if (type === 'uninstallVersion' && version) {
    emit('start-task', `Uninstalling ${name}@${version}`);
    try {
      if (sdkDetails.value[name]?.current === version) {
        await handleUnuse(name);
      }
      await UninstallVersion(name, version);
      await fetchDetail(name);
      await fetchVfoxSdks();
      delete versionPaths.value[version];
    } catch (err) {
      console.error(err);
      errorMsg.value = `Failed to uninstall ${name}@${version}.`;
    }
  } else if (type === 'removePlugin') {
    emit('start-task', `Removing plugin: ${name}`);
    removingPlugin.value = name;
    try {
      await handleUnuse(name);
      await RestoreSystemPath(name);
      await RemovePlugin(name);
      await fetchAllSdks();
      closeDetail();
    } catch (err) {
      console.error(err);
      errorMsg.value = `Failed to remove plugin ${name}.`;
    } finally {
      removingPlugin.value = null;
    }
  } else if (type === 'removeCustomSdk' && path) {
    const isCurrent = activeCustomSdk.value === path;
    if (isCurrent) {
      await handleUnuse(name);
    }
    await handleRemoveCustomPath(name, path);
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
    console.error(err);
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
    await ClipboardSetText(path);
    copiedPath.value = path;
    setTimeout(() => {
      if (copiedPath.value === path) {
        copiedPath.value = null;
      }
    }, 2000);
  }
};

const expandedVersions = ref<Record<string, boolean>>({});
const toggleExpand = (id: string) => {
  expandedVersions.value[id] = !expandedVersions.value[id];
};
</script>

<template>
  <div class="sdk-manager">
    <div v-if="errorMsg" class="error-banner" @click="errorMsg = ''">{{ errorMsg }}</div>

    <!-- MAIN VIEW (LIST) -->
    <Transition :name="transitionName" mode="out-in">
      <div v-if="activeView === 'list'" key="list" class="view-container">
        <h2>{{ t('sdk.installed.title') }}</h2>
        
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
                  <span class="version-count" :title="sdk.versions?.[0]?.version || 'unknown'">{{ (sdk.versions?.[0]?.version || 'unknown').length > 30 ? (sdk.versions?.[0]?.version || 'unknown').substring(0, 30) + '...' : (sdk.versions?.[0]?.version || 'unknown') }}</span>
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
                  v-else-if="!isWin11CompatApplied" 
                  class="btn tonal small" 
                  @click="handleHijackPlugin(selectedSdk.name)" 
                  :disabled="hijacking || restoring" 
                  style="min-width: 140px; display: flex; justify-content: center; align-items: center;"
                >
                  <div v-if="hijacking" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px;"></div>
                  <template v-else>
                    <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">security</span>
                    {{ t('sdk.add_system_path') }}
                    <div class="custom-tooltip-container" style="margin-left: 6px; display: flex;">
                      <span class="material-symbols-outlined" style="font-size: 14px; color: var(--md-outline); cursor: help;">info</span>
                      <div class="custom-tooltip-content">
                        {{ t('sdk.add_system_path.tooltip') }}
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
                    {{ t('sdk.remove_system_path') }}
                    <div class="custom-tooltip-container" style="margin-left: 6px; display: flex;">
                      <span class="material-symbols-outlined" style="font-size: 14px; color: var(--md-outline); cursor: help;">info</span>
                      <div class="custom-tooltip-content">
                        {{ t('sdk.remove_system_path.tooltip') }}
                      </div>
                    </div>
                  </template>
                </button>
                <span v-if="!checkingCompat && !isWin11CompatApplied" style="font-size: 11px; color: var(--md-outline); opacity: 0.8;">{{ t('sdk.add_system_path.hint') }}</span>
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
                  <div class="flex-center flex-gap-12" style="min-width: 0; flex: 1;">
                    <h3 class="version-title" :style="{ display: 'flex', alignItems: expandedVersions['sys-' + selectedSdk.path] ? 'flex-start' : 'center', flexWrap: expandedVersions['sys-' + selectedSdk.path] ? 'wrap' : 'nowrap', gap: '8px', minWidth: 0, flex: 1 }">
                      <span :style="{ wordBreak: 'break-all', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: expandedVersions['sys-' + selectedSdk.path] ? 'normal' : 'nowrap' }">
                        {{ selectedSdk.versions?.[0]?.version || 'unknown' }}
                      </span>
                      <button v-if="(selectedSdk.versions?.[0]?.version || 'unknown').length > 50" class="btn text small" style="padding: 0 4px; min-width: auto; height: 20px; line-height: 20px; flex-shrink: 0;" @click="toggleExpand('sys-' + selectedSdk.path)">
                        {{ expandedVersions['sys-' + selectedSdk.path] ? '收起' : '展开' }}
                      </button>
                    </h3>
                    <span class="system-tag" style="flex-shrink: 0;">{{ t('sdk.custom') }}</span>
                  </div>
                </div>
                <div class="version-card-body">
                  <div class="path-label">{{ t('sdk.exe_path') }}</div>
                  <div class="path-box">
                    <code class="path-text">{{ selectedSdk.path }}</code>
                    <button class="btn icon-btn" @click="copyPath(selectedSdk.path)" title="Copy path">
                      <svg v-if="copiedPath === selectedSdk.path" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
                      <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                    </button>
                  </div>
                  <p class="empty-state" style="margin-top: 12px; font-size: 14px; text-align: left;">
                    To manage this SDK, install the vfox plugin from Plugin Market first.
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
                    <div class="flex-center flex-gap-12" style="min-width: 0; flex: 1;">
                      <h3 class="version-title" :style="{ display: 'flex', alignItems: expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? 'flex-start' : 'center', flexWrap: expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? 'wrap' : 'nowrap', gap: '8px', minWidth: 0, flex: 1 }">
                        <span :style="{ wordBreak: 'break-all', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? 'normal' : 'nowrap' }">
                          {{ ver.version }}
                        </span>
                        <button v-if="ver.version.length > 50" class="btn text small" style="padding: 0 4px; min-width: auto; height: 20px; line-height: 20px; flex-shrink: 0;" @click="toggleExpand(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)">
                          {{ expandedVersions[(ver.isCustom ? 'sys-' + ver.path : 'vfox-' + ver.version)] ? '收起' : '展开' }}
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
                      <button class="btn icon-btn" @click="copyPath(ver.path)" title="Copy path">
                        <svg v-if="copiedPath === ver.path" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
                        <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                      </button>
                    </div>
                    <div v-else class="path-box loading-path">
                      <div class="spinner small"></div> Loading path...
                    </div>
                  </div>
                </div>
              </div>

              <div v-else class="empty-state">No versions installed.</div>

              <!-- Install Section (vfox only) -->
              <div v-if="selectedSdk.source === 'vfox'" class="install-section-large" style="margin-top: 24px; padding-top: 24px; border-top: 1px dashed var(--md-sys-color-outline-variant);">
                <button v-if="searchingFor !== selectedSdk.name" class="btn primary large-btn" @click="handleSearch(selectedSdk.name)">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                  Install New Version
                </button>

                <div v-else class="search-box large">
                  <div class="search-header">
                    <input v-model="searchQuery" type="text" class="search-input" placeholder="Search available versions..." autofocus />
                    <button class="btn text" @click="searchingFor = null">Cancel</button>
                  </div>
                  <div v-if="searchLoading" class="flex-center" style="padding: 24px;"><div class="spinner"></div></div>
                  <div v-else class="search-results-grid">
                    <button v-for="ver in filteredSearchResults" :key="ver" class="search-result-card" @click="handleInstall(selectedSdk.name, ver)">
                      {{ ver }}
                      <span class="install-text">Install</span>
                    </button>
                    <div v-if="!filteredSearchResults.length" class="empty-state" style="grid-column: 1/-1;">No matching versions found.</div>
                  </div>
                </div>
              </div>

              <!-- Add Custom Path Form -->
              <div class="install-section-large" style="margin-top: 24px; padding-top: 24px; border-top: 1px dashed var(--md-sys-color-outline-variant);">
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
        :title="confirmAction.type === 'removePlugin' ? t('sdk.remove_plugin') : confirmAction.type === 'uninstallVersion' ? t('sdk.uninstall') : t('sdk.remove')"
        :message="confirmAction.type === 'removePlugin' 
          ? `${t('sdk.confirm.remove_plugin')} '${confirmAction.name}'?` 
          : confirmAction.type === 'uninstallVersion'
          ? `${t('sdk.confirm.uninstall_version')} ${confirmAction.version} of ${confirmAction.name}?`
          : `${t('sdk.confirm.note')} ${t('sdk.confirm.remove_custom')}?`"
        :danger="true"
        :confirmText="confirmAction.type === 'removePlugin' ? t('sdk.remove') : confirmAction.type === 'uninstallVersion' ? t('sdk.uninstall') : t('sdk.remove_reference')"
        @confirm="executeConfirm"
        @cancel="confirmAction.type = null"
      />
    </Teleport>
  </div>
</template>
