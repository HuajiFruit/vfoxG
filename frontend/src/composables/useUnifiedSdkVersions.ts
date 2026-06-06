import type { SdkDetail, SdkInfo } from '../services/appModels';
import { computed, type Ref } from 'vue';
import type { UnifiedSdkVersion } from '../components/sdk/ManagedSdkDetailView.vue';

type UseUnifiedSdkVersionsOptions = {
  selectedSdk: Ref<SdkInfo | null>;
  sdkDetails: Ref<Record<string, SdkDetail>>;
  versionPaths: Ref<Record<string, Record<string, string>>>;
  nonVfoxSdksMap: Ref<Record<string, SdkInfo[]>>;
  activeCustomSdk: Ref<string>;
};

export const useUnifiedSdkVersions = (options: UseUnifiedSdkVersionsOptions) => {
  const matchingNonVfoxSdks = computed(() => {
    if (!options.selectedSdk.value) return [];
    return options.nonVfoxSdksMap.value[options.selectedSdk.value.name] || [];
  });

  const unifiedVersions = computed<UnifiedSdkVersion[]>(() => {
    if (!options.selectedSdk.value) return [];
    const sdkName = options.selectedSdk.value.name;
    const versions: UnifiedSdkVersion[] = [];
    const detail = options.sdkDetails.value[sdkName];

    for (const version of detail?.versions || []) {
      versions.push({
        isCustom: false,
        version: version.version,
        path: options.versionPaths.value[sdkName]?.[version.version] || '',
        isCurrent: version.isCurrent,
        vfoxVersion: version.version,
      });
    }

    for (const systemSdk of matchingNonVfoxSdks.value) {
      versions.push({
        isCustom: true,
        version: systemSdk.versions?.[0]?.version || 'unknown',
        path: systemSdk.path,
        isCurrent: (options.activeCustomSdk.value || '').toLowerCase() === (systemSdk.path || '').toLowerCase(),
        sysSdk: systemSdk,
      });
    }

    return versions;
  });

  return {
    matchingNonVfoxSdks,
    unifiedVersions,
  };
};
