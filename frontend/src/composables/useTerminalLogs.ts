import { nextTick, ref, type Ref } from 'vue';
import { getTerminalLogLevel } from './taskLogParser';
import type { TerminalLogEntry } from './taskTerminalTypes';

type UseTerminalLogsOptions = {
  showTerminal: Ref<boolean>;
  scrollTerminalToBottom: () => void;
};

const maxTerminalLogs = 500;

const getTerminalTimestamp = () => new Date().toLocaleTimeString([], {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
});

export const useTerminalLogs = (options: UseTerminalLogsOptions) => {
  const terminalLogs = ref<TerminalLogEntry[]>([]);
  let terminalLogId = 0;

  const appendTerminalLog = (log: string) => {
    terminalLogs.value.push({
      id: terminalLogId++,
      level: getTerminalLogLevel(log),
      text: log,
      time: getTerminalTimestamp(),
    });
    if (terminalLogs.value.length > maxTerminalLogs) {
      terminalLogs.value.splice(0, terminalLogs.value.length - maxTerminalLogs);
    }
    if (options.showTerminal.value) {
      nextTick(options.scrollTerminalToBottom);
    }
  };

  const clearTerminalLogs = () => {
    terminalLogs.value = [];
  };

  return {
    terminalLogs,
    appendTerminalLog,
    clearTerminalLogs,
  };
};
