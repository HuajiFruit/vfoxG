import type { TaskPhase, TerminalLogLevel } from './taskTerminalTypes';

const taskErrorPrefixes = [
  '[EXIT ERROR]',
  '[TIMEOUT]',
  '[APP ERROR]',
  '[STDOUT READ ERROR]',
  '[STDERR READ ERROR]',
];

export const isTaskDoneLog = (log: string) => log.startsWith('[DONE]');

export const isTaskErrorLog = (log: string) => taskErrorPrefixes.some(prefix => log.startsWith(prefix));

export const isVersionNotReleasedMessage = (message: string) => {
  const text = message.toLowerCase();
  return text.includes('version is not released') ||
    message.includes('版本未发布') ||
    message.includes('无发布版本');
};

export const getTerminalLogLevel = (log: string): TerminalLogLevel => {
  if (isTaskErrorLog(log)) return 'error';
  if (isTaskDoneLog(log)) return 'success';
  return 'info';
};

export const extractProgressPercent = (log: string): number | null => {
  const matches = [...log.matchAll(/(\d{1,3})(?:\.\d+)?\s*%/g)];
  if (!matches.length) return null;
  const latestPercent = Number(matches[matches.length - 1][1]);
  if (Number.isNaN(latestPercent)) return null;
  return Math.max(0, Math.min(100, latestPercent));
};

export const extractDownloadSpeed = (log: string): string => {
  const matches = [...log.matchAll(/(\d+(?:\.\d+)?)\s*([KMGT]?i?B|[KMGT]?B|bytes?)\s*\/\s*s(?:ec(?:ond)?|econd)?/gi)];
  if (!matches.length) return '';
  const latestSpeed = matches[matches.length - 1];
  const speedValue = Number(latestSpeed[1]);
  if (!Number.isFinite(speedValue) || speedValue <= 0) return '';
  const speedUnit = latestSpeed[2].replace(/^b$/i, 'B');
  return `${latestSpeed[1]} ${speedUnit}/s`;
};

export const getTaskPhaseFromLog = (log: string): TaskPhase | null => {
  const text = log.toLowerCase();
  if (
    text.includes('download') ||
    text.includes('fetch') ||
    text.includes('downloading') ||
    text.includes('下载')
  ) {
    return 'download';
  }
  if (
    text.includes('install') ||
    text.includes('extract') ||
    text.includes('unpack') ||
    text.includes('link') ||
    text.includes('安装') ||
    text.includes('解压')
  ) {
    return 'install';
  }
  return null;
};

export const getInitialTaskPhase = (title: string): TaskPhase => getTaskPhaseFromLog(title) || 'default';

export const isDownloadThenInstallTask = (title: string) => {
  const text = title.toLowerCase();
  return (
    text.includes('installing') ||
    text.includes('importing sdk') ||
    title.includes('安装') ||
    title.includes('导入 SDK')
  );
};

export const getRawTaskErrorMessage = (log: string) => log
  .replace(/^\[(?:EXIT ERROR|STDOUT READ ERROR|STDERR READ ERROR|TIMEOUT|APP ERROR)\]\s*/, '')
  .trim();
