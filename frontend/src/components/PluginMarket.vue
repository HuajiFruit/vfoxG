<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { GetAvailablePlugins, RefreshAvailablePlugins, RunVfoxWithProgress, SearchVersions, GetSdkDetail, InstallVersion, RemovePlugin, GetAddedPlugins, ScanSystemSdks, GetCachedSystemSdks, DetectSdkPathVersion, AddNonVfoxSdk, UseCustomSdk, HijackSystemPath, RestoreSystemPath } from '../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { main } from '../../wailsjs/go/models';
import PluginIcon from './PluginIcon.vue';
import ConfirmModal from './ConfirmModal.vue';
import { t } from '../i18n';
import { getPluginDescription } from '../pluginDescriptions';

const plugins = ref<main.PluginInfo[]>([]);
const loading = ref(true);
const addingPlugin = ref<string | null>(null);
const removingPlugin = ref<string | null>(null);

// Computed groups
const officialPlugins = computed(() => plugins.value.filter(p => p.isOfficial));
const communityPlugins = computed(() => plugins.value.filter(p => !p.isOfficial));

// Master-Detail State
const activeView = ref<'list' | 'detail'>('list');
const transitionName = ref('fade-slide-forward');
const selectedPlugin = ref<main.PluginInfo | null>(null);

// Detail State
const availableVersions = ref<string[]>([]);
const installedVersions = ref<Set<string>>(new Set());
const loadingVersions = ref(false);
const installingVersion = ref<string | null>(null);

const emit = defineEmits(['start-task']);

const fetchPlugins = async () => {
  if (plugins.value.length === 0) {
    loading.value = true;
  }
  try {
    const [available, addedPlugins] = await Promise.all([
      GetAvailablePlugins(),
      GetAddedPlugins()
    ]);
    const addedNames = new Set(addedPlugins);
    plugins.value = available.map(p => ({
      ...p,
      isAdded: addedNames.has(p.name)
    }));
    loading.value = false;

    // Background silent refresh
    RefreshAvailablePlugins().then(async () => {
      const [freshAvailable, freshAdded] = await Promise.all([
        GetAvailablePlugins(),
        GetAddedPlugins()
      ]);
      const freshAddedNames = new Set(freshAdded);
      plugins.value = freshAvailable.map(p => ({
        ...p,
        isAdded: freshAddedNames.has(p.name)
      }));
      if (selectedPlugin.value) {
        const updated = plugins.value.find(p => p.name === selectedPlugin.value!.name);
        if (updated) {
          selectedPlugin.value = updated;
        }
      }
    }).catch(e => console.error(e));
    
    // Update selectedPlugin if we are in detail view
    if (selectedPlugin.value) {
      const updated = plugins.value.find(p => p.name === selectedPlugin.value!.name);
      if (updated) {
        selectedPlugin.value = updated;
      }
    }
  } catch (err) {
    console.error(err);
  } finally {
    loading.value = false;
  }
};

const openUrl = (url: string) => {
  BrowserOpenURL(url);
};

const fetchPluginVersions = async (name: string) => {
  loadingVersions.value = true;
  availableVersions.value = [];
  installedVersions.value.clear();
  try {
    const results = await SearchVersions(name);
    availableVersions.value = results;
    
    try {
      const detail = await GetSdkDetail(name);
      const instSet = new Set<string>();
      if (detail && detail.versions) {
        detail.versions.forEach(v => instSet.add(v.version));
      }
      installedVersions.value = instSet;
    } catch (e) {
      // SDK might not have any installed versions, that's fine
    }
  } catch (err) {
    console.error(err);
  } finally {
    loadingVersions.value = false;
  }
};

const openDetail = async (p: main.PluginInfo) => {
  selectedPlugin.value = p;
  transitionName.value = 'fade-slide-forward';
  activeView.value = 'detail';
  
  if (p.isAdded) {
    await fetchPluginVersions(p.name);
  }
};

const closeDetail = () => {
  transitionName.value = 'fade-slide-backward';
  activeView.value = 'list';
  setTimeout(() => {
    selectedPlugin.value = null;
    availableVersions.value = [];
    installedVersions.value.clear();
  }, 300);
};

const addPlugin = async (name: string) => {
  emit('start-task', `Adding plugin: ${name}`);
  addingPlugin.value = name;
  try {
    await RunVfoxWithProgress(['add', name]);
    await fetchPlugins(); // This will also update selectedPlugin.isAdded
    
    // Auto-add and use system SDK if it exists
    try {
      const systemSdks = await GetCachedSystemSdks();
      const matchingSdk = systemSdks?.find(s => s.name === name);
      if (matchingSdk && matchingSdk.path) {
        const version = await DetectSdkPathVersion(name, matchingSdk.path);
        await AddNonVfoxSdk(name, matchingSdk.path, version || 'unknown');
        await UseCustomSdk(name, matchingSdk.path);
        // Hijack the original system path to fulfill the aggressive unset behavior requirement
        await HijackSystemPath(name, matchingSdk.path);
      }
    } catch (e) {
      console.error("Failed to auto-add system SDK:", e);
    }

    ScanSystemSdks(); // Update system SDK cache in background
    
    if (selectedPlugin.value && selectedPlugin.value.name === name) {
      await fetchPluginVersions(name);
    }
  } catch (err) {
    console.error(err);
  } finally {
    if (addingPlugin.value === name) {
      addingPlugin.value = null;
    }
  }
};

const confirmRemove = ref<string | null>(null);

const requestRemovePlugin = (name: string) => {
  confirmRemove.value = name;
};

const executeRemovePlugin = async () => {
  if (!confirmRemove.value) return;
  const name = confirmRemove.value;
  confirmRemove.value = null;
  
  emit('start-task', `Removing plugin: ${name}`);
  removingPlugin.value = name;
  try {
    await RestoreSystemPath(name);
    await RemovePlugin(name);
    await fetchPlugins();
    ScanSystemSdks(); // Update system SDK cache in background // This updates the global list and selectedPlugin
  } catch (err) {
    console.error(err);
  } finally {
    removingPlugin.value = null;
  }
};

const installVersion = async (pluginName: string, version: string) => {
  emit('start-task', `Installing ${pluginName}@${version}`);
  installingVersion.value = version;
  try {
    await InstallVersion(pluginName, version);
    installedVersions.value.add(version);
  } catch (err) {
    console.error(err);
  } finally {
    if (installingVersion.value === version) {
      installingVersion.value = null;
    }
  }
};

onMounted(() => {
  fetchPlugins();
});
</script>

<template>
  <div class="plugin-market">
    
    <Transition :name="transitionName" mode="out-in">
      <!-- MAIN VIEW (LIST) -->
      <div v-if="activeView === 'list'" key="list" class="view-container">
        <h2>{{ t('market.title') }}</h2>

        <div v-if="loading" class="flex-center" style="height: 200px; flex-direction: column; gap: 16px;">
          <div class="spinner"></div>
          <span style="color: var(--text-secondary); font-size: 14px;">{{ t('market.loading') }}</span>
        </div>

        <div v-else>
          <!-- Official Plugins -->
          <h3 class="section-heading" v-if="officialPlugins.length > 0">Official Plugins</h3>
          <div class="sdk-grid">
            <div v-for="p in officialPlugins" :key="p.name" class="sdk-card" @click="openDetail(p)">
              <PluginIcon :name="p.name" class="sdk-icon-large" />
              <div class="sdk-card-content" style="flex: 1;">
                <h3>{{ p.name }}</h3>
                <button class="link" style="font-size: 12px; margin-top: 4px; padding: 0;" @click.stop="openUrl(p.url)">{{ t('market.homepage') }} &nearr;</button>
              </div>
              <div class="plugin-actions">
                <span v-if="p.isAdded" class="btn tonal small" style="pointer-events: none; width: 80px; padding: 0; display: flex; justify-content: center; align-items: center; text-transform: uppercase; font-weight: 600;">{{ t('market.installed') }}</span>
                <button 
                  v-else 
                  class="btn primary small" 
                  :disabled="addingPlugin === p.name"
                  @click.stop="addPlugin(p.name)"
                  style="width: 80px; padding: 0; display: flex; justify-content: center; align-items: center;"
                >
                  <div v-if="addingPlugin === p.name" class="spinner small-spinner" style="width: 14px; height: 14px; border-width: 2px;"></div>
                  <template v-else>{{ t('sdk.add') }}</template>
                </button>
              </div>
            </div>
          </div>

          <!-- Community Plugins -->
          <h3 class="section-heading" v-if="communityPlugins.length > 0" style="margin-top: 32px; color: var(--text-secondary);">Community Plugins</h3>
          <div class="sdk-grid" v-if="communityPlugins.length > 0">
            <div v-for="p in communityPlugins" :key="p.name" class="sdk-card" @click="openDetail(p)">
              <PluginIcon :name="p.name" class="sdk-icon-large" />
              <div class="sdk-card-content" style="flex: 1;">
                <h3>{{ p.name }}</h3>
                <button class="link" style="font-size: 12px; margin-top: 4px; padding: 0;" @click.stop="openUrl(p.url)">{{ t('market.homepage') }} &nearr;</button>
              </div>
              <div class="plugin-actions">
                <span v-if="p.isAdded" class="btn tonal small" style="pointer-events: none; width: 80px; padding: 0; display: flex; justify-content: center; align-items: center; text-transform: uppercase; font-weight: 600;">{{ t('market.installed') }}</span>
                <button 
                  v-else 
                  class="btn primary small" 
                  :disabled="addingPlugin === p.name"
                  @click.stop="addPlugin(p.name)"
                  style="width: 80px; padding: 0; display: flex; justify-content: center; align-items: center;"
                >
                  <div v-if="addingPlugin === p.name" class="spinner small-spinner" style="width: 14px; height: 14px; border-width: 2px;"></div>
                  <template v-else>{{ t('sdk.add') }}</template>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- SECONDARY PAGE (DETAIL) -->
      <div v-else-if="activeView === 'detail' && selectedPlugin" key="detail" class="view-container detail-view">
        <div class="detail-header">
          <button class="btn tonal small back-btn" @click="closeDetail">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
            {{ t('sdk.back') }}
          </button>
          
          <div class="detail-title-area" style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
            <div style="display: flex; align-items: center; gap: 24px;">
              <PluginIcon :name="selectedPlugin.name" class="sdk-icon-huge" />
              <div class="detail-title-text">
                <h1>{{ selectedPlugin.name }}</h1>
                <p v-if="getPluginDescription(selectedPlugin.name)" style="color: var(--text-secondary); margin: 4px 0 8px 0; font-size: 14px; max-width: 500px; line-height: 1.5;">{{ getPluginDescription(selectedPlugin.name) }}</p>
                <button class="link" @click="openUrl(selectedPlugin.url)">{{ t('market.homepage') }} &nearr;</button>
              </div>
            </div>
            
            <div v-if="selectedPlugin.isAdded">
              <button 
                class="btn tonal small" 
                style="color: var(--md-error); background: rgba(239, 68, 68, 0.1); min-width: 120px; display: flex; justify-content: center; align-items: center;" 
                :disabled="removingPlugin === selectedPlugin.name"
                @click="requestRemovePlugin(selectedPlugin.name)"
              >
                <div v-if="removingPlugin === selectedPlugin.name" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px; border-top-color: var(--md-error);"></div>
                <template v-else>
                  <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">delete</span>
                  {{ t('sdk.remove_plugin') }}
                </template>
              </button>
            </div>
          </div>
        </div>

        <div class="detail-body">
          <div v-if="!selectedPlugin.isAdded" class="plugin-not-added-banner flex-center" style="flex-direction: column; padding: 60px 20px; text-align: center;">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="1.5" style="margin-bottom: 16px;"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
            <h2 style="margin-bottom: 8px;">Plugin Not Added</h2>
            <p style="color: var(--text-secondary); margin-bottom: 24px; max-width: 400px;">
              You need to add this plugin to your vfox environment before you can browse and install its available SDK versions.
            </p>
            <button class="btn primary large-btn" :disabled="addingPlugin === selectedPlugin.name" @click="addPlugin(selectedPlugin.name)" style="min-width: 140px; display: flex; justify-content: center; align-items: center;">
              <div v-if="addingPlugin === selectedPlugin.name" class="spinner small-spinner" style="width: 18px; height: 18px; border-width: 2px;"></div>
              <template v-else>+ {{ t('sdk.add') }} Plugin</template>
            </button>
          </div>

          <div v-else>
            <h2>{{ t('sdk.available_versions') }}</h2>
            <p style="color: var(--text-secondary); margin-bottom: 20px;">Select a version below to install it locally.</p>

            <div v-if="loadingVersions" class="flex-center" style="height: 200px;">
              <div class="spinner"></div>
            </div>
            
            <div v-else-if="availableVersions.length === 0" class="empty-state">
              No versions found for this plugin.
            </div>

            <div v-else class="search-results-grid">
              <div v-for="ver in availableVersions" :key="ver" class="search-result-card" style="cursor: default;">
                <span>{{ ver }}</span>
                <div class="plugin-actions">
                  <span v-if="installedVersions.has(ver)" class="btn tonal small" style="pointer-events: none; width: 96px; padding: 0; display: flex; justify-content: center; align-items: center; color: #34d399; font-weight: 600; text-transform: uppercase;">{{ t('market.installed') }}</span>
                  <button 
                    v-else 
                    class="btn tonal small" 
                    :disabled="installingVersion === ver"
                    @click="installVersion(selectedPlugin.name, ver)"
                    style="width: 96px; padding: 0;"
                  >
                    {{ installingVersion === ver ? '...' : t('market.install') }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>

    <Teleport to="body">
      <ConfirmModal 
        v-if="confirmRemove"
        :title="t('sdk.remove_plugin')"
        :message="`${t('sdk.confirm.remove_plugin')} '${confirmRemove}'?`"
        :danger="true"
        :confirmText="t('sdk.remove')"
        @confirm="executeRemovePlugin"
        @cancel="confirmRemove = null"
      />
    </Teleport>
  </div>
</template>
