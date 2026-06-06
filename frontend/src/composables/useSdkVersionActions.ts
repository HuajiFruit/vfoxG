import type { SdkDetail, SdkInfo } from '../services/appModels';
import { ref, type Ref } from 'vue';
import { t } from '../i18n';
import {
  installVfoxVersion,
  uninstallVfoxVersion,
  unuseSdk,
  useNonVfoxSdk,
  useVfoxVersion,
} from '../services/sdkManagerService';

type RunTask = (title: string, task: () => Promise<void>) => Promise<boolean>;

type UseSdkVersionActionsOptions = {
  selectedSdk: Ref<SdkInfo | null>;
  sdkDetails: Ref<Record<string, SdkDetail>>;
  activeCustomSdk: Ref<string>;
  runTask: RunTask;
  notifyError: (message: string) => void;
  notifyTaskError: (err: unknown, fallback: string) => void;
  formatError: (err: unknown, fallback: string) => string;
  refreshSdkDetail: (sdkName: string) => Promise<void>;
  fetchVfoxSdks: () => Promise<void>;
  checkCompatMode: (sdkName: string) => Promise<void>;
  clearSdkCurrentVersion: (sdkName: string) => void;
  clearActiveCustomSdk: (sdkName: string) => void;
  fetchSingleVersionPath: (sdkName: string, version: string) => Promise<void>;
  removeVersionPath: (sdkName: string, version: string) => void;
};

export const useSdkVersionActions = (options: UseSdkVersionActionsOptions) => {
  const usingVersion = ref<string | null>(null);
  const unusingSdk = ref<string | null>(null);

  const clearCurrentState = (sdkName: string) => {
    options.clearSdkCurrentVersion(sdkName);
    options.clearActiveCustomSdk(sdkName);
  };

  const refreshSelectedSdkState = async (sdkName: string) => {
    await options.refreshSdkDetail(sdkName);
    await options.fetchVfoxSdks();
    await options.checkCompatMode(sdkName);
  };

  const handleUseCustomPath = async (sdkName: string, customSdkPath: string) => {
    usingVersion.value = customSdkPath;
    try {
      await options.runTask(t('task.custom.use', { name: sdkName, path: customSdkPath }), async () => {
        const result = await useNonVfoxSdk(sdkName, customSdkPath);
        if (result !== 'ok') {
          options.notifyError(t('sdk.custom.apply_error'));
        }
        clearCurrentState(sdkName);
        await refreshSelectedSdkState(sdkName);
      });
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.custom.apply_exception')));
    } finally {
      usingVersion.value = null;
    }
  };

  const handleUse = async (sdkName: string, version: string) => {
    usingVersion.value = version;
    try {
      await options.runTask(t('task.version.switch', { name: sdkName, version }), async () => {
        await useVfoxVersion(sdkName, version);
        await refreshSelectedSdkState(sdkName);
      });
    } catch (err) {
      options.notifyTaskError(err, t('sdk.switch_error', { name: sdkName, version }));
    } finally {
      usingVersion.value = null;
    }
  };

  const handleUnuse = async (sdkName: string) => {
    if (unusingSdk.value === sdkName) return;
    unusingSdk.value = sdkName;
    try {
      await options.runTask(t('task.version.unset', { name: sdkName }), async () => {
        await unuseSdk(sdkName);
        clearCurrentState(sdkName);
        await options.fetchVfoxSdks();
        if (options.selectedSdk.value?.name === sdkName) {
          await options.refreshSdkDetail(sdkName);
          await options.checkCompatMode(sdkName);
        }
      });
    } catch (err) {
      options.notifyTaskError(err, t('sdk.unset_error', { name: sdkName }));
    } finally {
      if (unusingSdk.value === sdkName) {
        unusingSdk.value = null;
      }
    }
  };

  const handleInstall = async (sdkName: string, version: string) => {
    try {
      await options.runTask(t('task.version.install', { name: sdkName, version }), async () => {
        await installVfoxVersion(sdkName, version);
        await options.refreshSdkDetail(sdkName);
        await options.fetchVfoxSdks();
        await options.fetchSingleVersionPath(sdkName, version);
      });
    } catch (err) {
      options.notifyTaskError(err, t('sdk.install_error', { name: sdkName, version }));
    }
  };

  const handleUninstall = async (sdkName: string, version: string) => {
    try {
      await options.runTask(t('task.version.uninstall', { name: sdkName, version }), async () => {
        if (options.sdkDetails.value[sdkName]?.current === version) {
          await unuseSdk(sdkName);
          clearCurrentState(sdkName);
        }
        await uninstallVfoxVersion(sdkName, version);
        await options.refreshSdkDetail(sdkName);
        await options.fetchVfoxSdks();
        options.removeVersionPath(sdkName, version);
      });
    } catch (err) {
      options.notifyTaskError(err, t('sdk.uninstall_error', { name: sdkName, version }));
    }
  };

  const removeCurrentCustomSdkBeforeDelete = async (sdkName: string, customSdkPath: string) => {
    if (options.activeCustomSdk.value === customSdkPath) {
      await handleUnuse(sdkName);
    }
  };

  return {
    usingVersion,
    unusingSdk,
    clearCurrentState,
    handleUseCustomPath,
    handleUse,
    handleUnuse,
    handleInstall,
    handleUninstall,
    removeCurrentCustomSdkBeforeDelete,
  };
};
