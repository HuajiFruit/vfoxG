import { ref } from 'vue';
import { t } from '../i18n';
import { fetchNonVfoxSdks, removePluginWithChoice } from '../services/sdkManagerService';

type ConfirmAction = {
  type: 'removePlugin' | 'uninstallVersion' | 'removeCustomSdk' | null;
  name: string;
  version?: string;
  path?: string;
};

type UseSdkRemovalDialogsOptions = {
  runTask: (title: string, task: () => Promise<void>) => Promise<boolean>;
  notifyError: (message: string) => void;
  notifyTaskError: (err: unknown, fallback: string) => void;
  formatError: (err: unknown, fallback: string) => string;
  handleUninstall: (sdkName: string, version: string) => Promise<void>;
  removeCurrentCustomSdkBeforeDelete: (sdkName: string, customSdkPath: string) => Promise<void>;
  handleRemoveCustomPath: (sdkName: string, customSdkPath: string) => Promise<void>;
  fetchAllSdks: () => Promise<void>;
  closeDetail: () => void;
};

export const useSdkRemovalDialogs = (options: UseSdkRemovalDialogsOptions) => {
  const removingPlugin = ref<string | null>(null);
  const confirmAction = ref<ConfirmAction>({ type: null, name: '' });
  const removePluginName = ref<string | null>(null);
  const removePluginCustomSdks = ref<Array<{ path: string; version: string }>>([]);

  const requestUninstall = (sdkName: string, version: string) => {
    confirmAction.value = { type: 'uninstallVersion', name: sdkName, version };
  };

  const requestRemoveCustomPath = (sdkName: string, customSdkPath: string) => {
    confirmAction.value = { type: 'removeCustomSdk', name: sdkName, path: customSdkPath };
  };

  const requestRemovePlugin = async (sdkName: string) => {
    try {
      const nonVfoxMap = await fetchNonVfoxSdks();
      const customSdks = nonVfoxMap[sdkName] || [];
      removePluginCustomSdks.value = customSdks.map((sdk: any) => ({
        path: sdk.path || sdk.Path || '',
        version: sdk.versions?.[0]?.version || sdk.version || sdk.Version || 'unknown',
      }));
    } catch (err) {
      removePluginCustomSdks.value = [];
      options.notifyError(options.formatError(err, t('market.custom_refs_error', { name: sdkName })));
    }
    removePluginName.value = sdkName;
  };

  const executeConfirm = async () => {
    const { type, name, version, path } = confirmAction.value;
    confirmAction.value = { type: null, name: '' };

    if (type === 'uninstallVersion' && version) {
      await options.handleUninstall(name, version);
      return;
    }
    if (type === 'removeCustomSdk' && path) {
      await options.removeCurrentCustomSdkBeforeDelete(name, path);
      await options.handleRemoveCustomPath(name, path);
    }
  };

  const executeRemovePlugin = async (chosenPath: string | null) => {
    const sdkName = removePluginName.value;
    removePluginName.value = null;
    if (!sdkName) return;

    removingPlugin.value = sdkName;
    try {
      await options.runTask(t('task.plugin.remove', { name: sdkName }), async () => {
        await removePluginWithChoice(sdkName, chosenPath || '');
        await options.fetchAllSdks();
        options.closeDetail();
      });
    } catch (err) {
      options.notifyTaskError(err, t('sdk.remove_plugin_error', { name: sdkName }));
    } finally {
      removingPlugin.value = null;
    }
  };

  return {
    removingPlugin,
    confirmAction,
    removePluginName,
    removePluginCustomSdks,
    requestUninstall,
    requestRemoveCustomPath,
    requestRemovePlugin,
    executeConfirm,
    executeRemovePlugin,
  };
};
