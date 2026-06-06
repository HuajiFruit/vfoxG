import type { PlatformInfo, SdkInfo } from '../services/appModels';
import { computed, ref, type Ref } from 'vue';
import { t } from '../i18n';
import {
  checkPluginPathOverride,
  disablePluginPathOverride,
  enablePluginPathOverride,
  fetchActiveCustomSdk,
  fetchPlatformInfo,
} from '../services/sdkManagerService';

type UsePathOverrideActionsOptions = {
  selectedSdk: Ref<SdkInfo | null>;
  notifyError: (message: string) => void;
  notifySuccess: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

export const usePathOverrideActions = (options: UsePathOverrideActionsOptions) => {
  const platformInfo = ref<PlatformInfo | null>(null);
  const checkingCompat = ref(false);
  const isPathOverrideApplied = ref(false);
  const activeCustomSdk = ref('');
  const hijacking = ref(false);
  const restoring = ref(false);
  let compatRequestSeq = 0;

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

  const invalidateCompatRequests = () => {
    compatRequestSeq++;
  };

  const clearActiveCustomSdk = (sdkName: string) => {
    if (options.selectedSdk.value?.name === sdkName) {
      activeCustomSdk.value = '';
    }
  };

  const loadPlatformInfo = async () => {
    try {
      platformInfo.value = await fetchPlatformInfo();
    } catch (err) {
      options.notifyError(options.formatError(err, t('settings.platform.load_error')));
    }
  };

  const checkCompatMode = async (sdkName: string) => {
    const requestId = ++compatRequestSeq;
    checkingCompat.value = true;
    try {
      const applied = await checkPluginPathOverride(sdkName);
      const activeSdk = await fetchActiveCustomSdk(sdkName);
      if (requestId !== compatRequestSeq || options.selectedSdk.value?.name !== sdkName) return;
      isPathOverrideApplied.value = applied;
      activeCustomSdk.value = activeSdk;
    } catch (err) {
      if (requestId === compatRequestSeq && options.selectedSdk.value?.name === sdkName) {
        options.notifyError(options.formatError(err, t('sdk.path_override.check_error', { name: sdkName })));
      }
    } finally {
      if (requestId === compatRequestSeq) {
        checkingCompat.value = false;
      }
    }
  };

  const handleHijackPlugin = async (sdkName: string) => {
    hijacking.value = true;
    try {
      await enablePluginPathOverride(sdkName);
      await checkCompatMode(sdkName);
      options.notifySuccess(t('sdk.path_override.enable_success', {
        name: sdkName,
        restart: pathOverrideRestartHint.value,
      }));
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.path_override.enable_error', { name: sdkName })));
    } finally {
      hijacking.value = false;
    }
  };

  const handleRestorePlugin = async (sdkName: string) => {
    restoring.value = true;
    try {
      await disablePluginPathOverride(sdkName);
      await checkCompatMode(sdkName);
      options.notifySuccess(t('sdk.path_override.disable_success', {
        name: sdkName,
        restart: pathOverrideRestartHint.value,
      }));
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.path_override.disable_error', { name: sdkName })));
    } finally {
      restoring.value = false;
    }
  };

  return {
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
  };
};
