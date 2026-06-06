import type { PluginInfo } from '../services/appModels';
import { ref } from 'vue';
import { t } from '../i18n';
import {
  fetchSdkDetail,
  fetchSearchVersions,
} from '../services/pluginMarketService';

type UsePluginVersionDetailOptions = {
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

export const usePluginVersionDetail = (options: UsePluginVersionDetailOptions) => {
  const activeView = ref<'list' | 'detail'>('list');
  const transitionName = ref('fade-slide-forward');
  const selectedPlugin = ref<PluginInfo | null>(null);
  const availableVersions = ref<string[]>([]);
  const installedVersions = ref<Set<string>>(new Set());
  const loadingVersions = ref(false);
  let versionsFetchSeq = 0;
  let detailResetTimer: ReturnType<typeof setTimeout> | null = null;

  const resetVersionDetail = () => {
    availableVersions.value = [];
    installedVersions.value = new Set();
    loadingVersions.value = false;
  };

  const clearDetailResetTimer = () => {
    if (detailResetTimer !== null) {
      clearTimeout(detailResetTimer);
      detailResetTimer = null;
    }
  };

  const invalidateVersionRequests = () => {
    versionsFetchSeq++;
  };

  const fetchPluginVersions = async (pluginName: string) => {
    const requestId = ++versionsFetchSeq;
    resetVersionDetail();
    loadingVersions.value = true;
    try {
      const results = await fetchSearchVersions(pluginName);
      if (requestId !== versionsFetchSeq) return;
      availableVersions.value = results;

      try {
        const detail = await fetchSdkDetail(pluginName);
        if (requestId !== versionsFetchSeq) return;
        installedVersions.value = new Set((detail?.versions || []).map(version => version.version));
      } catch {
        installedVersions.value = new Set();
      }
    } catch (err) {
      if (requestId === versionsFetchSeq) {
        options.notifyError(options.formatError(err, t('market.versions_error', { name: pluginName })));
      }
    } finally {
      if (requestId === versionsFetchSeq) {
        loadingVersions.value = false;
      }
    }
  };

  const openDetail = async (plugin: PluginInfo) => {
    clearDetailResetTimer();
    invalidateVersionRequests();
    selectedPlugin.value = plugin;
    transitionName.value = 'fade-slide-forward';
    activeView.value = 'detail';
    resetVersionDetail();

    if (plugin.isAdded) {
      await fetchPluginVersions(plugin.name);
    }
  };

  const closeDetail = () => {
    clearDetailResetTimer();
    invalidateVersionRequests();
    transitionName.value = 'fade-slide-backward';
    activeView.value = 'list';
    detailResetTimer = setTimeout(() => {
      selectedPlugin.value = null;
      resetVersionDetail();
      detailResetTimer = null;
    }, 300);
  };

  const resetDetailImmediately = () => {
    clearDetailResetTimer();
    invalidateVersionRequests();
    selectedPlugin.value = null;
    resetVersionDetail();
    activeView.value = 'list';
  };

  const disposePluginVersionDetail = () => {
    clearDetailResetTimer();
    invalidateVersionRequests();
  };

  return {
    activeView,
    transitionName,
    selectedPlugin,
    availableVersions,
    installedVersions,
    loadingVersions,
    fetchPluginVersions,
    openDetail,
    closeDetail,
    resetDetailImmediately,
    disposePluginVersionDetail,
  };
};
