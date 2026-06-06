import { ref } from 'vue';
import { t } from '../i18n';
import { copyTextToClipboard } from '../services/sdkManagerService';

type UseCopyablePathOptions = {
  notifyInfo: (message: string) => void;
  notifyError: (message: string) => void;
  formatError: (err: unknown, fallback: string) => string;
};

export const useCopyablePath = (options: UseCopyablePathOptions) => {
  const copiedPath = ref<string | null>(null);

  const copyPath = async (path: string) => {
    if (!path) return;

    try {
      await copyTextToClipboard(path);
      copiedPath.value = path;
      options.notifyInfo(t('sdk.path.copied'));
      setTimeout(() => {
        if (copiedPath.value === path) {
          copiedPath.value = null;
        }
      }, 2000);
    } catch (err) {
      options.notifyError(options.formatError(err, t('sdk.path.copy_error')));
    }
  };

  return {
    copiedPath,
    copyPath,
  };
};
