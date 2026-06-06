import type { SdkDetail, SdkInfo, SdkVersionDetail } from '../services/appModels';
import { ref, type Ref } from 'vue';
import { t } from '../i18n';
import { fetchSdkDetail, fetchVersionPath } from '../services/sdkManagerService';

type UseSdkDetailDataOptions = {
  selectedSdk: Ref<SdkInfo | null>;
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

export const useSdkDetailData = (options: UseSdkDetailDataOptions) => {
  const sdkDetails = ref<Record<string, SdkDetail>>({});
  const detailError = ref<Record<string, boolean>>({});
  const versionPaths = ref<Record<string, Record<string, string>>>({});
  let detailRequestSeq = 0;

  const beginDetailRequest = (sdkName?: string) => {
    const requestId = ++detailRequestSeq;
    if (sdkName) {
      versionPaths.value[sdkName] = {};
    }
    return requestId;
  };

  const invalidateDetailRequests = () => {
    detailRequestSeq++;
  };

  const isCurrentDetailRequest = (sdkName: string, requestId: number) => (
    requestId === detailRequestSeq &&
    options.selectedSdk.value?.name === sdkName
  );

  const fetchDetail = async (sdkName: string, requestId = beginDetailRequest()) => {
    detailError.value[sdkName] = false;
    try {
      const sdkDetail = await fetchSdkDetail(sdkName);
      if (requestId !== detailRequestSeq) return;
      sdkDetails.value[sdkName] = sdkDetail;
    } catch (err) {
      if (requestId === detailRequestSeq) {
        detailError.value[sdkName] = true;
        options.notifyError(options.formatError(err, t('sdk.detail_error', { name: sdkName })));
      }
    }
  };

  const refreshSdkDetail = async (sdkName: string) => {
    await fetchDetail(sdkName, beginDetailRequest());
  };

  const fetchDetailVersionPaths = async (sdkName: string, requestId: number) => {
    const detailVersions = sdkDetails.value[sdkName]?.versions || [];
    for (const version of detailVersions) {
      try {
        const sdkVersionPath = await fetchVersionPath(sdkName, version.version);
        if (!isCurrentDetailRequest(sdkName, requestId)) return;
        versionPaths.value[sdkName][version.version] = sdkVersionPath;
      } catch (err) {
        if (requestId === detailRequestSeq) {
          options.notifyError(options.formatError(err, t('sdk.path.load_error', {
            name: sdkName,
            version: version.version,
          })));
        }
      }
    }
  };

  const fetchSingleVersionPath = async (sdkName: string, version: string) => {
    const sdkVersionPath = await fetchVersionPath(sdkName, version);
    versionPaths.value[sdkName] ||= {};
    versionPaths.value[sdkName][version] = sdkVersionPath;
  };

  const clearSdkCurrentVersion = (sdkName: string) => {
    const sdkDetail = sdkDetails.value[sdkName];
    if (!sdkDetail) return;

    sdkDetail.current = '';
    (sdkDetail.versions || []).forEach((version: SdkVersionDetail) => {
      version.isCurrent = false;
    });
  };

  const removeVersionPath = (sdkName: string, version: string) => {
    const sdkVersionPaths = versionPaths.value[sdkName];
    if (!sdkVersionPaths) return;

    delete sdkVersionPaths[version];
    if (Object.keys(sdkVersionPaths).length === 0) {
      delete versionPaths.value[sdkName];
    }
  };

  return {
    sdkDetails,
    detailError,
    versionPaths,
    beginDetailRequest,
    invalidateDetailRequests,
    isCurrentDetailRequest,
    fetchDetail,
    refreshSdkDetail,
    fetchDetailVersionPaths,
    fetchSingleVersionPath,
    clearSdkCurrentVersion,
    removeVersionPath,
  };
};
