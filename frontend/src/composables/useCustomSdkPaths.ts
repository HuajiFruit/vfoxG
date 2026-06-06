import { ref } from 'vue';
import { t } from '../i18n';
import {
  addNonVfoxSdk,
  detectSdkVersion,
  removeNonVfoxSdk,
} from '../services/sdkManagerService';

type UseCustomSdkPathsOptions = {
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
  refreshNonVfoxSdks: () => Promise<unknown>;
  checkCompatMode: (sdkName: string) => Promise<void>;
};

export const useCustomSdkPaths = (options: UseCustomSdkPathsOptions) => {
  const customPathInput = ref('');
  const customVersionInput = ref('');
  const detectingCustomVersion = ref(false);
  const isAddingCustomPath = ref(false);
  const isAddingPathMode = ref<string | null>(null);

  const resetCustomPathForm = () => {
    customPathInput.value = '';
    customVersionInput.value = '';
  };

  const handleDetectVersion = async (sdkName: string) => {
    const customSdkPath = customPathInput.value.trim();
    if (!customSdkPath || detectingCustomVersion.value) return;

    detectingCustomVersion.value = true;
    try {
      const detectedVersion = await detectSdkVersion(sdkName, customSdkPath);
      if (detectedVersion && detectedVersion !== 'unknown') {
        customVersionInput.value = detectedVersion;
      }
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.detect_error')));
    } finally {
      detectingCustomVersion.value = false;
    }
  };

  const handleAddCustomPath = async (sdkName: string) => {
    const customSdkPath = customPathInput.value.trim();
    if (!customSdkPath) return;

    isAddingCustomPath.value = true;
    try {
      await addNonVfoxSdk(sdkName, customSdkPath, customVersionInput.value.trim());
      resetCustomPathForm();
      isAddingPathMode.value = null;
      await options.refreshNonVfoxSdks();
      await options.checkCompatMode(sdkName);
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.custom.add_error')));
    } finally {
      isAddingCustomPath.value = false;
    }
  };

  const handleRemoveCustomPath = async (sdkName: string, customSdkPath: string) => {
    try {
      await removeNonVfoxSdk(sdkName, customSdkPath);
      await options.refreshNonVfoxSdks();
      await options.checkCompatMode(sdkName);
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.custom.remove_error')));
    }
  };

  const startAddCustomPath = (sdkName: string) => {
    isAddingPathMode.value = sdkName;
  };

  const cancelAddCustomPath = () => {
    isAddingPathMode.value = null;
    resetCustomPathForm();
  };

  return {
    customPathInput,
    customVersionInput,
    isAddingCustomPath,
    isAddingPathMode,
    handleDetectVersion,
    handleAddCustomPath,
    handleRemoveCustomPath,
    startAddCustomPath,
    cancelAddCustomPath,
  };
};
