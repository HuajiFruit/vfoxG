import type { SdkDetail, SdkVersionDetail } from '../services/appModels';
import { computed, ref, type Ref } from 'vue';
import { t } from '../i18n';
import { searchSdkVersions } from '../services/sdkManagerService';
import { formatVersionTitle, normalizeVersionKey } from './sdkVersionFormat';

type UseSdkVersionSearchOptions = {
  sdkDetails: Ref<Record<string, SdkDetail>>;
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

export const useSdkVersionSearch = (options: UseSdkVersionSearchOptions) => {
  const searchingFor = ref<string | null>(null);
  const searchResults = ref<string[]>([]);
  const searchLoading = ref(false);
  const searchQuery = ref('');

  const filteredSearchResults = computed(() => {
    if (!searchQuery.value) return searchResults.value;
    const query = searchQuery.value.toLowerCase();
    return searchResults.value.filter(version => version.toLowerCase().includes(query));
  });

  const isSearchVersionInstalled = (sdkName: string, version: string) => {
    const targetVersion = normalizeVersionKey(formatVersionTitle(sdkName, version));
    if (!targetVersion) return false;
    const detailVersions = options.sdkDetails.value[sdkName]?.versions || [];
    return detailVersions.some((detailVersion: SdkVersionDetail) =>
      normalizeVersionKey(formatVersionTitle(sdkName, detailVersion.version)) === targetVersion);
  };

  const handleSearch = async (sdkName: string) => {
    searchingFor.value = sdkName;
    searchQuery.value = '';
    searchLoading.value = true;
    try {
      const results = await searchSdkVersions(sdkName);
      if (searchingFor.value === sdkName) {
        searchResults.value = results;
      }
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.search_error', { name: sdkName })));
      if (searchingFor.value === sdkName) {
        searchResults.value = [];
      }
    } finally {
      if (searchingFor.value === sdkName) {
        searchLoading.value = false;
      }
    }
  };

  const cancelSearch = () => {
    searchingFor.value = null;
  };

  const resetSearch = () => {
    searchingFor.value = null;
    searchResults.value = [];
    searchQuery.value = '';
  };

  return {
    searchingFor,
    searchLoading,
    searchQuery,
    filteredSearchResults,
    isSearchVersionInstalled,
    handleSearch,
    cancelSearch,
    resetSearch,
  };
};
