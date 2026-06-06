import { computed, onMounted, ref, watch } from 'vue';
import { t } from '../i18n';
import {
  PendingMigrationAction,
  shouldAskDownloadPathMigration,
} from './downloadPathMigration';
import {
  DownloadPathInfo,
  fetchDownloadPathInfo,
  resetDownloadPath,
  resetDownloadPathWithMigration,
  saveDownloadPath,
  saveDownloadPathWithMigration,
  selectDownloadPath,
} from '../services/settingsService';
import {
  SettingsNotice,
  getErrorMessage,
  useSettingsNotice,
} from './useSettingsNotice';

export const useDownloadPathSettings = (notify: (notice: SettingsNotice) => void) => {
  const downloadPathInfo = ref<DownloadPathInfo | null>(null);
  const downloadPathInput = ref('');
  const loadingDownloadPath = ref(true);
  const savingDownloadPath = ref(false);
  const selectingDownloadPath = ref(false);
  const resettingDownloadPath = ref(false);
  const pendingMigration = ref<PendingMigrationAction | null>(null);
  const { notifyError, notifySuccess } = useSettingsNotice(notify);

  const syncDownloadPathInfo = (info: DownloadPathInfo) => {
    downloadPathInfo.value = info;
    downloadPathInput.value = info.path;
  };

  const shouldAskMigration = (targetPath: string) => {
    return shouldAskDownloadPathMigration(downloadPathInfo.value, targetPath);
  };

  const isDownloadPathBusy = computed(() => savingDownloadPath.value || resettingDownloadPath.value);

  const loadDownloadPath = async () => {
    loadingDownloadPath.value = true;
    try {
      syncDownloadPathInfo(await fetchDownloadPathInfo());
    } catch (err) {
      notifyError(getErrorMessage(err, t('settings.download.path.load_error')));
    } finally {
      loadingDownloadPath.value = false;
    }
  };

  const requestSaveDownloadPath = async () => {
    const targetPath = downloadPathInput.value.trim();
    if (shouldAskMigration(targetPath)) {
      pendingMigration.value = { type: 'save', targetPath };
      return;
    }

    pendingMigration.value = null;
    savingDownloadPath.value = true;
    try {
      syncDownloadPathInfo(await saveDownloadPath(targetPath));
      notifySuccess(t('settings.download.path.success'));
    } catch (err) {
      notifyError(getErrorMessage(err, t('settings.download.path.error')));
    } finally {
      savingDownloadPath.value = false;
    }
  };

  const chooseDownloadPath = async () => {
    selectingDownloadPath.value = true;
    try {
      const selected = await selectDownloadPath();
      if (selected && selected.trim()) {
        downloadPathInput.value = selected;
      }
    } catch (err) {
      notifyError(getErrorMessage(err, t('settings.download.path.select_error')));
    } finally {
      selectingDownloadPath.value = false;
    }
  };

  const requestResetDownloadPath = async () => {
    const targetPath = downloadPathInfo.value?.defaultPath || '';
    if (shouldAskMigration(targetPath)) {
      pendingMigration.value = { type: 'reset', targetPath };
      return;
    }

    pendingMigration.value = null;
    resettingDownloadPath.value = true;
    try {
      syncDownloadPathInfo(await resetDownloadPath());
      notifySuccess(t('settings.download.path.reset_success'));
    } catch (err) {
      notifyError(getErrorMessage(err, t('settings.download.path.error')));
    } finally {
      resettingDownloadPath.value = false;
    }
  };

  const confirmPendingMigration = async () => {
    const action = pendingMigration.value;
    if (!action || isDownloadPathBusy.value) return;

    if (action.type === 'save') {
      savingDownloadPath.value = true;
      try {
        syncDownloadPathInfo(await saveDownloadPathWithMigration(action.targetPath));
        pendingMigration.value = null;
        notifySuccess(t('settings.download.path.migrate_success'));
      } catch (err) {
        notifyError(getErrorMessage(err, t('settings.download.path.error')));
      } finally {
        savingDownloadPath.value = false;
      }
      return;
    }

    resettingDownloadPath.value = true;
    try {
      syncDownloadPathInfo(await resetDownloadPathWithMigration());
      pendingMigration.value = null;
      notifySuccess(t('settings.download.path.migrate_success'));
    } catch (err) {
      notifyError(getErrorMessage(err, t('settings.download.path.error')));
    } finally {
      resettingDownloadPath.value = false;
    }
  };

  const cancelPendingMigration = () => {
    if (isDownloadPathBusy.value) return;
    pendingMigration.value = null;
  };

  watch(downloadPathInput, (value) => {
    if (pendingMigration.value?.type === 'save' && value.trim() !== pendingMigration.value.targetPath) {
      pendingMigration.value = null;
    }
  });

  onMounted(() => {
    loadDownloadPath();
  });

  return {
    downloadPathInfo,
    downloadPathInput,
    loadingDownloadPath,
    savingDownloadPath,
    selectingDownloadPath,
    resettingDownloadPath,
    pendingMigration,
    isDownloadPathBusy,
    chooseDownloadPath,
    requestSaveDownloadPath,
    requestResetDownloadPath,
    confirmPendingMigration,
    cancelPendingMigration,
  };
};
