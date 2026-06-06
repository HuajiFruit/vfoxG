import { onMounted, onUnmounted } from 'vue';
import { onRuntimeEvent } from '../services/runtimeService';

type UsePluginMarketLifecycleOptions = {
  fetchPlugins: () => Promise<void>;
  invalidatePluginRequests: () => void;
  resetDetailImmediately: () => void;
  disposePluginVersionDetail: () => void;
};

export const usePluginMarketLifecycle = (options: UsePluginMarketLifecycleOptions) => {
  let vfoxHomeChangedOff: (() => void) | null = null;

  onMounted(() => {
    options.fetchPlugins();
    vfoxHomeChangedOff = onRuntimeEvent<void>('vfox-home-changed', () => {
      options.invalidatePluginRequests();
      options.resetDetailImmediately();
      options.fetchPlugins();
    });
  });

  onUnmounted(() => {
    options.invalidatePluginRequests();
    options.disposePluginVersionDetail();
    if (vfoxHomeChangedOff) {
      vfoxHomeChangedOff();
      vfoxHomeChangedOff = null;
    }
  });
};
