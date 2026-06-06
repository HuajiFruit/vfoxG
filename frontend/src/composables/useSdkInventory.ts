import type { SdkInfo } from '../services/appModels';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { onRuntimeEvent } from '../services/runtimeService';
import {
  fetchAllSdks as fetchAllSdksFromBackend,
  fetchInstalledSdks,
  fetchNonVfoxSdks,
} from '../services/sdkManagerService';

type UseSdkInventoryOptions = {
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
  sdkRefreshError: string;
  sdkLoadError: string;
  onSdkListChanged?: () => void;
};

const safeSdkList = (value: SdkInfo[] | null | undefined) => Array.isArray(value) ? value : [];

const mergeSdkLists = (vfox: SdkInfo[] | null | undefined, system: SdkInfo[] | null | undefined) => {
  vfox = safeSdkList(vfox);
  system = safeSdkList(system);
  const vfoxNames = new Set(vfox.map(sdk => sdk.name));
  return [...vfox, ...system.filter(sdk => !vfoxNames.has(sdk.name))];
};

export const useSdkInventory = (options: UseSdkInventoryOptions) => {
  const sdks = ref<SdkInfo[]>([]);
  const loading = ref(true);
  const nonVfoxSdksMap = ref<Record<string, SdkInfo[]>>({});
  let sdksFetchSeq = 0;
  let systemReadyOff: (() => void) | null = null;

  const vfoxSdks = computed(() => sdks.value.filter(sdk => sdk.source === 'vfox'));
  const systemSdks = computed(() => sdks.value.filter(sdk => sdk.source !== 'vfox'));

  const fetchVfoxSdks = async () => {
    try {
      const vfox = await fetchInstalledSdks();
      const system = sdks.value.filter(sdk => sdk.source !== 'vfox');
      sdks.value = mergeSdkLists(vfox, system);
    } catch (err) {
      options.notifyError(options.formatError(err, options.sdkRefreshError));
    }
  };

  const fetchAllSdks = async () => {
    const requestId = ++sdksFetchSeq;
    loading.value = true;
    try {
      const all = await fetchAllSdksFromBackend();
      if (requestId !== sdksFetchSeq) return;
      sdks.value = safeSdkList(all);
    } catch (err) {
      if (requestId === sdksFetchSeq) {
        options.notifyError(options.formatError(err, options.sdkLoadError));
      }
    } finally {
      if (requestId === sdksFetchSeq) {
        loading.value = false;
      }
    }
  };

  const refreshNonVfoxSdks = async () => {
    nonVfoxSdksMap.value = await fetchNonVfoxSdks();
    return nonVfoxSdksMap.value;
  };

  onMounted(async () => {
    await fetchAllSdks();
    let mounted = true;

    const offSystem = onRuntimeEvent<void>('system-sdks-ready', () => {
      if (!mounted) return;
      fetchAllSdksFromBackend().then(all => {
        if (mounted) sdks.value = safeSdkList(all);
      }).catch(() => {});
    });

    const offSdkChanged = onRuntimeEvent<void>('sdk-list-changed', () => {
      if (!mounted) return;
      fetchAllSdks();
      refreshNonVfoxSdks().catch(() => {});
      options.onSdkListChanged?.();
    });

    refreshNonVfoxSdks().catch(() => {});

    systemReadyOff = () => {
      mounted = false;
      offSystem();
      offSdkChanged();
    };
  });

  onUnmounted(() => {
    if (systemReadyOff) systemReadyOff();
  });

  return {
    sdks,
    loading,
    nonVfoxSdksMap,
    vfoxSdks,
    systemSdks,
    fetchAllSdks,
    fetchVfoxSdks,
    refreshNonVfoxSdks,
  };
};
