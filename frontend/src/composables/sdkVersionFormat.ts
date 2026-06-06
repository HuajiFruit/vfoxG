import { t } from '../i18n';

export const formatVersionTitle = (sdkName: string, rawVersion: string) => {
  if (!rawVersion) return 'unknown';
  const sdkNamePrefix = new RegExp(`^${sdkName}\\s*`, 'i');
  return rawVersion.replace(sdkNamePrefix, '').trim();
};

export const displayVersion = (version?: string) => version || t('common.unknown');

export const truncateVersion = (version?: string, maxLength = 30) => {
  const text = displayVersion(version);
  return text.length > maxLength ? `${text.substring(0, maxLength)}...` : text;
};

export const normalizeVersionKey = (version?: string) => (version || '')
  .trim()
  .replace(/^v/i, '')
  .toLowerCase();
