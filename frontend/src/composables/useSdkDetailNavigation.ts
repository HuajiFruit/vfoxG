import type { SdkInfo } from '../services/appModels';
import { ref, type Ref } from 'vue';
import { t } from '../i18n';

type UseSdkDetailNavigationOptions = {
  selectedSdk: Ref<SdkInfo | null>;
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
  beginDetailRequest: (sdkName?: string) => number;
  invalidateDetailRequests: () => void;
  invalidateCompatRequests: () => void;
  isCurrentDetailRequest: (sdkName: string, requestId: number) => boolean;
  fetchDetail: (sdkName: string, requestId?: number) => Promise<void>;
  fetchDetailVersionPaths: (sdkName: string, requestId: number) => Promise<void>;
  refreshNonVfoxSdks: () => Promise<unknown>;
  checkCompatMode: (sdkName: string) => Promise<void>;
  resetSearch: () => void;
};

export const useSdkDetailNavigation = (options: UseSdkDetailNavigationOptions) => {
  const activeView = ref<'list' | 'detail'>('list');
  const transitionName = ref('fade-slide-forward');
  let detailResetTimer: ReturnType<typeof setTimeout> | null = null;

  const clearDetailResetTimer = () => {
    if (detailResetTimer !== null) {
      clearTimeout(detailResetTimer);
      detailResetTimer = null;
    }
  };

  const openDetail = async (sdk: SdkInfo) => {
    clearDetailResetTimer();
    const requestId = options.beginDetailRequest(sdk.name);
    options.selectedSdk.value = sdk;
    transitionName.value = 'fade-slide-forward';
    activeView.value = 'detail';

    await options.fetchDetail(sdk.name, requestId);
    if (!options.isCurrentDetailRequest(sdk.name, requestId)) return;
    await options.fetchDetailVersionPaths(sdk.name, requestId);

    try {
      await options.refreshNonVfoxSdks();
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.custom_paths.load_error')));
    }

    await options.checkCompatMode(sdk.name);
  };

  const closeDetail = () => {
    clearDetailResetTimer();
    options.invalidateDetailRequests();
    options.invalidateCompatRequests();
    transitionName.value = 'fade-slide-backward';
    activeView.value = 'list';
    detailResetTimer = setTimeout(() => {
      options.selectedSdk.value = null;
      options.resetSearch();
      detailResetTimer = null;
    }, 300);
  };

  return {
    activeView,
    transitionName,
    clearDetailResetTimer,
    openDetail,
    closeDetail,
  };
};
