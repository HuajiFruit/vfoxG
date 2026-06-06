import type { PluginInfo } from '../services/appModels';
import { ref, type Ref } from 'vue';
import { t } from '../i18n';
import {
  addCustomSdkReference,
  addVfoxPlugin,
  detectSystemSdkVersion,
  fetchCachedSystemSdks,
  fetchNonVfoxSdks,
  hijackSystemSdkPath,
  installPluginVersion,
  refreshSystemSdkCache,
  removePlugin,
  useCustomSdkReference,
} from '../services/pluginMarketService';

type UsePluginMarketActionsOptions = {
  selectedPlugin: Ref<PluginInfo | null>;
  installedVersions: Ref<Set<string>>;
  runTask: (title: string, task: () => Promise<void>) => Promise<boolean>;
  notifyError: (message: string) => void;
  notifyTaskError: (err: unknown, fallback: string) => void;
  formatError: (err: unknown, fallback: string) => string;
  fetchPlugins: () => Promise<void>;
  fetchPluginVersions: (pluginName: string) => Promise<void>;
};

export const usePluginMarketActions = (options: UsePluginMarketActionsOptions) => {
  const addingPlugin = ref<string | null>(null);
  const removingPlugin = ref<string | null>(null);
  const installingVersion = ref<string | null>(null);
  const confirmRemove = ref<string | null>(null);
  const confirmRemoveCustomSdks = ref<Array<{ path: string; version: string }>>([]);

  const linkSystemSdkAfterAdd = async (pluginName: string) => {
    try {
      const systemSdks = await fetchCachedSystemSdks();
      const matchingSdk = systemSdks?.find(sdk => sdk.name === pluginName);
      if (!matchingSdk?.path) return;

      const version = await detectSystemSdkVersion(pluginName, matchingSdk.path);
      await addCustomSdkReference(pluginName, matchingSdk.path, version || 'unknown');
      await useCustomSdkReference(pluginName, matchingSdk.path);
      await hijackSystemSdkPath(pluginName, matchingSdk.path);
    } catch (err) {
      options.notifyError(options.formatError(err, t('market.add_auto_link_error', { name: pluginName })));
    }
  };

  const addPlugin = async (pluginName: string) => {
    addingPlugin.value = pluginName;
    try {
      await options.runTask(t('task.plugin.add', { name: pluginName }), async () => {
        await addVfoxPlugin(pluginName);
        await options.fetchPlugins();
        await linkSystemSdkAfterAdd(pluginName);
        refreshSystemSdkCache();

        if (options.selectedPlugin.value?.name === pluginName) {
          await options.fetchPluginVersions(pluginName);
        }
      });
    } catch (err) {
      options.notifyTaskError(err, t('market.add_error', { name: pluginName }));
    } finally {
      if (addingPlugin.value === pluginName) {
        addingPlugin.value = null;
      }
    }
  };

  const requestRemovePlugin = async (pluginName: string) => {
    try {
      const nonVfoxMap = await fetchNonVfoxSdks();
      const customSdks = nonVfoxMap[pluginName] || [];
      confirmRemoveCustomSdks.value = customSdks.map((sdk: any) => ({
        path: sdk.path || sdk.Path || '',
        version: sdk.versions?.[0]?.version || sdk.version || sdk.Version || 'unknown',
      }));
    } catch (err) {
      confirmRemoveCustomSdks.value = [];
      options.notifyError(options.formatError(err, t('market.custom_refs_error', { name: pluginName })));
    }
    confirmRemove.value = pluginName;
  };

  const executeRemovePlugin = async (chosenPath: string | null) => {
    if (!confirmRemove.value) return;
    const pluginName = confirmRemove.value;
    confirmRemove.value = null;

    removingPlugin.value = pluginName;
    try {
      await options.runTask(t('task.plugin.remove', { name: pluginName }), async () => {
        await removePlugin(pluginName, chosenPath || '');
        await options.fetchPlugins();
        refreshSystemSdkCache();
      });
    } catch (err) {
      options.notifyTaskError(err, t('market.remove_error', { name: pluginName }));
    } finally {
      removingPlugin.value = null;
    }
  };

  const installVersion = async (pluginName: string, version: string) => {
    installingVersion.value = version;
    try {
      await options.runTask(t('task.version.install', { name: pluginName, version }), async () => {
        await installPluginVersion(pluginName, version);
        await options.fetchPlugins();
        if (options.selectedPlugin.value?.name === pluginName) {
          await options.fetchPluginVersions(pluginName);
        }
        options.installedVersions.value.add(version);
      });
    } catch (err) {
      options.notifyTaskError(err, t('market.install_error', { name: pluginName, version }));
    } finally {
      if (installingVersion.value === version) {
        installingVersion.value = null;
      }
    }
  };

  return {
    addingPlugin,
    removingPlugin,
    installingVersion,
    confirmRemove,
    confirmRemoveCustomSdks,
    addPlugin,
    requestRemovePlugin,
    executeRemovePlugin,
    installVersion,
  };
};
