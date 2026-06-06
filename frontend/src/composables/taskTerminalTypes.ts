export type ToastStatus = 'running' | 'success' | 'error' | 'info';

export type NotifyPayload = string | {
  title?: string;
  message: string;
  type?: Exclude<ToastStatus, 'running'>;
  durationMs?: number;
};

export type TaskPhase = 'default' | 'download' | 'install';

export type TerminalLogLevel = 'info' | 'success' | 'error';

export type TerminalLogEntry = {
  id: number;
  level: TerminalLogLevel;
  text: string;
  time: string;
};
