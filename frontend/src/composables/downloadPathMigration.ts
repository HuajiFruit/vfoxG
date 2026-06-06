import type { DownloadPathInfo } from '../services/settingsService';

export type PendingMigrationAction = {
  type: 'save' | 'reset';
  targetPath: string;
};

export const shouldAskDownloadPathMigration = (
  downloadPathInfo: DownloadPathInfo | null,
  targetPath: string
) => {
  const currentPath = downloadPathInfo?.path?.trim();
  const normalizedTargetPath = targetPath.trim();

  return Boolean(
    downloadPathInfo?.hasMigratableData &&
      currentPath &&
      normalizedTargetPath &&
      currentPath !== normalizedTargetPath
  );
};
