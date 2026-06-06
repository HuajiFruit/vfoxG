import type { PluginInfo } from '../services/appModels';
import { computed, ref, type Ref } from 'vue';
import { t } from '../i18n';
import {
  fetchAddedPlugins,
  fetchAvailablePlugins,
  refreshAvailablePlugins,
} from '../services/pluginMarketService';

type UsePluginMarketListOptions = {
  selectedPlugin: Ref<PluginInfo | null>;
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

const markAddedPlugins = (availablePlugins: PluginInfo[], addedPlugins: string[]) => {
  const addedNames = new Set(addedPlugins);
  return availablePlugins.map(plugin => ({
    ...plugin,
    isAdded: addedNames.has(plugin.name),
  }));
};

export const usePluginMarketList = (options: UsePluginMarketListOptions) => {
  const plugins = ref<PluginInfo[]>([]);
  const loading = ref(true);
  let pluginsFetchSeq = 0;

  const officialPlugins = computed(() => plugins.value.filter(plugin => plugin.isOfficial));
  const communityPlugins = computed(() => plugins.value.filter(plugin => !plugin.isOfficial));

  const syncSelectedPlugin = () => {
    if (!options.selectedPlugin.value) return;
    const updatedPlugin = plugins.value.find(plugin => plugin.name === options.selectedPlugin.value!.name);
    if (updatedPlugin) {
      options.selectedPlugin.value = updatedPlugin;
    }
  };

  const loadPluginSnapshot = async (requestId: number) => {
    const [availablePlugins, addedPlugins] = await Promise.all([
      fetchAvailablePlugins(),
      fetchAddedPlugins(),
    ]);
    if (requestId !== pluginsFetchSeq) return;
    plugins.value = markAddedPlugins(availablePlugins, addedPlugins);
    syncSelectedPlugin();
  };

  const refreshPluginsInBackground = (requestId: number) => {
    void refreshAvailablePlugins().then(async () => {
      await loadPluginSnapshot(requestId);
    }).catch(err => {
      if (requestId === pluginsFetchSeq) {
        options.notifyError(options.formatError(err, t('market.refresh_error')));
      }
    });
  };

  const fetchPlugins = async () => {
    const requestId = ++pluginsFetchSeq;
    if (plugins.value.length === 0) {
      loading.value = true;
    }
    try {
      await loadPluginSnapshot(requestId);
      refreshPluginsInBackground(requestId);
    } catch (err) {
      if (requestId === pluginsFetchSeq) {
        options.notifyError(options.formatError(err, t('market.load_error')));
      }
    } finally {
      if (requestId === pluginsFetchSeq) {
        loading.value = false;
      }
    }
  };

  const invalidatePluginRequests = () => {
    pluginsFetchSeq++;
  };

  return {
    plugins,
    loading,
    officialPlugins,
    communityPlugins,
    fetchPlugins,
    invalidatePluginRequests,
  };
};
