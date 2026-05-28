<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch, provide, readonly } from 'vue';
import { EventsOn, WindowSetLightTheme, WindowSetDarkTheme, WindowSetSystemDefaultTheme } from '../wailsjs/runtime/runtime';
import { t } from './i18n';
import SdkManager from './components/SdkManager.vue';
import SdkSync from './components/SdkSync.vue';
import PluginMarket from './components/PluginMarket.vue';
import Settings from './components/Settings.vue';

// --- Theme Management ---
const theme = ref(localStorage.getItem('vfox-theme') || 'auto');
provide('theme', theme);
const showTerminal = ref(localStorage.getItem('vfox-show-terminal') === 'true');
provide('showTerminal', showTerminal);

const applyTheme = () => {
  if (theme.value === 'auto') {
    const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
    WindowSetSystemDefaultTheme();
  } else {
    document.documentElement.setAttribute('data-theme', theme.value);
    if (theme.value === 'light') {
      WindowSetLightTheme();
    } else {
      WindowSetDarkTheme();
    }
  }
};

watch(theme, (newVal) => {
  localStorage.setItem('vfox-theme', newVal);
  applyTheme();
});

watch(showTerminal, (newVal) => {
  localStorage.setItem('vfox-show-terminal', String(newVal));
  if (newVal) {
    nextTick(() => {
      scrollTerminalToBottom();
    });
  }
});

const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
const handleSystemThemeChange = () => {
  if (theme.value === 'auto') applyTheme();
};
// ------------------------

const currentTab = ref('sdk');
const navTransition = ref('slide-up');
const showTaskToast = ref(false);
const terminalTaskRunning = ref(false);
const busyHintVisible = ref(false);
type SdkSidebarAction = { id: number; type: 'display' };
const sdkSidebarAction = ref<SdkSidebarAction | null>(null);
let sdkSidebarActionId = 0;
type ToastStatus = 'running' | 'success' | 'error' | 'info';
type NotifyPayload = string | {
  title?: string;
  message: string;
  type?: Exclude<ToastStatus, 'running'>;
  durationMs?: number;
};
const taskErrorPrefixes = [
  '[EXIT ERROR]',
  '[TIMEOUT]',
  '[APP ERROR]',
  '[STDOUT READ ERROR]',
  '[STDERR READ ERROR]',
];
const isTaskDoneLog = (log: string) => log.startsWith('[DONE]');
const isTaskErrorLog = (log: string) => taskErrorPrefixes.some(prefix => log.startsWith(prefix));
const hasActiveTaskToast = () => showTaskToast.value && taskStatus.value === 'running' && taskTitle.value.trim() !== '';
const getErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err.trim()) return err;
  return fallback;
};
let busyHintTimer: ReturnType<typeof setTimeout> | null = null;

const showBusyHint = () => {
  busyHintVisible.value = true;
  if (busyHintTimer) clearTimeout(busyHintTimer);
  busyHintTimer = setTimeout(() => {
    busyHintVisible.value = false;
    busyHintTimer = null;
  }, 1800);
};

const runTerminalTask = async (title: string, task: () => Promise<void>) => {
  if (terminalTaskRunning.value) {
    showTaskToast.value = true;
    showBusyHint();
    return false;
  }
  terminalTaskRunning.value = true;
  handleStartTask(title);
  try {
    await task();
    completeRunningTaskSuccess();
    return true;
  } catch (err) {
    completeRunningTaskError(getErrorMessage(err, t('toast.task_failed')));
    throw err;
  } finally {
    terminalTaskRunning.value = false;
  }
};

provide('terminalTaskRunning', readonly(terminalTaskRunning));
provide('runTerminalTask', runTerminalTask);

const switchTab = (tab: string) => {
  if (tab === currentTab.value) return;
  const tabOrder = ['sdk', 'sync', 'plugin', 'settings'];
  const currentIndex = tabOrder.indexOf(currentTab.value);
  const nextIndex = tabOrder.indexOf(tab);
  navTransition.value = nextIndex >= currentIndex ? 'slide-up' : 'slide-down';
  currentTab.value = tab;
};

const triggerSdkSidebarAction = (type: SdkSidebarAction['type']) => {
  if (currentTab.value !== 'sdk') {
    switchTab('sdk');
  }
  sdkSidebarAction.value = { id: ++sdkSidebarActionId, type };
};

const clearSdkSidebarAction = (id: number) => {
  if (sdkSidebarAction.value?.id === id) {
    sdkSidebarAction.value = null;
  }
};
const taskTitle = ref('');
const taskStatus = ref<ToastStatus>('running');
const lastLogLine = ref('');
const taskProgress = ref(0);
const hasTaskProgress = ref(false);
const taskHadError = ref(false);
type TaskPhase = 'default' | 'download' | 'install';
const taskPhase = ref<TaskPhase>('default');
const taskUsesDownloadThenInstall = ref(false);
let autoCloseTimer: ReturnType<typeof setTimeout> | null = null;
let offVfoxLog: (() => void) | null = null;
let offVfoxBusy: (() => void) | null = null;
type TerminalLogLevel = 'info' | 'success' | 'error';
type TerminalLogEntry = {
  id: number;
  level: TerminalLogLevel;
  text: string;
  time: string;
};

const terminalLogs = ref<TerminalLogEntry[]>([]);
const terminalBody = ref<HTMLElement | null>(null);
let terminalLogId = 0;
const maxTerminalLogs = 500;

const getTerminalLogLevel = (log: string): TerminalLogLevel => {
  if (isTaskErrorLog(log)) return 'error';
  if (isTaskDoneLog(log)) return 'success';
  return 'info';
};

const getTerminalTimestamp = () => new Date().toLocaleTimeString([], {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
});

const scrollTerminalToBottom = () => {
  if (!terminalBody.value) return;
  terminalBody.value.scrollTop = terminalBody.value.scrollHeight;
};

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
  if (showTerminal.value) {
    nextTick(() => {
      scrollTerminalToBottom();
    });
  }
};

const clearTerminalLogs = () => {
  terminalLogs.value = [];
};

const extractProgressPercent = (log: string): number | null => {
  const matches = [...log.matchAll(/(\d{1,3})(?:\.\d+)?\s*%/g)];
  if (!matches.length) return null;
  const latest = Number(matches[matches.length - 1][1]);
  if (Number.isNaN(latest)) return null;
  return Math.max(0, Math.min(100, latest));
};

const getTaskPhaseFromLog = (log: string): TaskPhase | null => {
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

const getInitialTaskPhase = (title: string): TaskPhase => {
  return getTaskPhaseFromLog(title) || 'default';
};

const isDownloadThenInstallTask = (title: string) => {
  const text = title.toLowerCase();
  return (
    text.includes('installing') ||
    text.includes('importing sdk') ||
    title.includes('安装') ||
    title.includes('导入 SDK')
  );
};

const clearTaskProgress = () => {
  taskProgress.value = 0;
  hasTaskProgress.value = false;
};

const enterInstallPhase = () => {
  taskPhase.value = 'install';
  clearTaskProgress();
};

const isDeterminateDownloadProgress = computed(() => (
  taskStatus.value === 'running' &&
  taskPhase.value === 'download' &&
  hasTaskProgress.value
));

const showToastProgress = computed(() => (
  taskStatus.value === 'running' ||
  taskStatus.value === 'success' ||
  (taskStatus.value === 'error' && hasTaskProgress.value)
));

const toastProgressStyle = computed(() => {
  if (isDeterminateDownloadProgress.value) {
    return { width: `${taskProgress.value}%` };
  }
  if (taskStatus.value === 'success') {
    return { width: '100%' };
  }
  if (taskStatus.value === 'error' && hasTaskProgress.value) {
    return { width: `${taskProgress.value}%` };
  }
  return undefined;
});

const handleStartTask = (title: string) => {
  taskTitle.value = title;
  lastLogLine.value = t('toast.starting');
  taskStatus.value = 'running';
  clearTaskProgress();
  taskHadError.value = false;
  taskUsesDownloadThenInstall.value = isDownloadThenInstallTask(title);
  taskPhase.value = getInitialTaskPhase(title);
  showTaskToast.value = true;
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
};

const scheduleToastClose = (durationMs: number) => {
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
  autoCloseTimer = setTimeout(() => {
    showTaskToast.value = false;
  }, durationMs);
};

const completeRunningTaskSuccess = () => {
  if (taskStatus.value !== 'running') return;
  taskStatus.value = 'success';
  lastLogLine.value = t('toast.completed');
  taskProgress.value = 100;
  hasTaskProgress.value = true;
  scheduleToastClose(2500);
};

const completeRunningTaskError = (message: string) => {
  if (taskStatus.value !== 'running') return;
  taskHadError.value = true;
  taskStatus.value = 'error';
  lastLogLine.value = message || t('toast.task_failed');
  scheduleToastClose(5000);
};

const handleNotify = (payload: NotifyPayload) => {
  const notification = typeof payload === 'string'
    ? { message: payload, type: 'info' as const }
    : { type: 'info' as const, ...payload };

  taskTitle.value = notification.title || (
    notification.type === 'error'
      ? t('common.error')
      : notification.type === 'success'
        ? t('common.success')
        : t('common.notification')
  );
  lastLogLine.value = notification.message;
  taskStatus.value = notification.type;
  taskProgress.value = notification.type === 'success' ? 100 : 0;
  hasTaskProgress.value = notification.type === 'success';
  taskHadError.value = notification.type === 'error';
  taskUsesDownloadThenInstall.value = false;
  taskPhase.value = 'default';
  showTaskToast.value = true;
  scheduleToastClose(notification.durationMs ?? (notification.type === 'error' ? 5000 : 3200));
};

const formatTaskError = (log: string) => log
  .replace(/^\[(?:EXIT ERROR|STDOUT READ ERROR|STDERR READ ERROR|TIMEOUT|APP ERROR)\]\s*/, '')
  .trim() || t('toast.task_failed');

const closeToast = () => {
  showTaskToast.value = false;
  if (autoCloseTimer) {
    clearTimeout(autoCloseTimer);
    autoCloseTimer = null;
  }
};

onMounted(() => {
  applyTheme();
  mediaQuery.addEventListener('change', handleSystemThemeChange);

  offVfoxLog = EventsOn('vfox-log', (log: string) => {
    appendTerminalLog(log);

    if (!terminalTaskRunning.value && !hasActiveTaskToast()) {
      if (isTaskErrorLog(log)) {
        handleNotify({
          type: 'error',
          title: t('common.error'),
          message: formatTaskError(log),
        });
      }
      return;
    }

    const parsedProgress = extractProgressPercent(log);
    const nextTaskPhase = getTaskPhaseFromLog(log);
    const shouldEnterInstallAfterDownload = taskUsesDownloadThenInstall.value && parsedProgress !== null && parsedProgress >= 100;
    let runningLogLine = log;

    if (taskStatus.value === 'running' && parsedProgress !== null && nextTaskPhase !== 'install' && !shouldEnterInstallAfterDownload) {
      taskPhase.value = 'download';
      taskProgress.value = parsedProgress;
      hasTaskProgress.value = true;
    }
    if (taskStatus.value === 'running' && (nextTaskPhase === 'install' || shouldEnterInstallAfterDownload)) {
      enterInstallPhase();
      runningLogLine = t('toast.installing_after_download');
    } else if (nextTaskPhase) {
      taskPhase.value = nextTaskPhase;
    }

    showTaskToast.value = true;

    // Determine status
    if (isTaskDoneLog(log)) {
      if (taskHadError.value) {
        scheduleToastClose(5000);
        return;
      }
      taskStatus.value = 'success';
      lastLogLine.value = t('toast.completed');
      taskProgress.value = 100;
      hasTaskProgress.value = true;
    } else if (isTaskErrorLog(log)) {
      taskHadError.value = true;
      taskStatus.value = 'error';
      lastLogLine.value = formatTaskError(log);
    } else if (!taskHadError.value) {
      lastLogLine.value = runningLogLine;
    }

    if (taskStatus.value !== 'running') {
      scheduleToastClose(taskStatus.value === 'error' ? 5000 : 2500); // Wait a bit longer to show success/error state before closing
    }
  });

  offVfoxBusy = EventsOn('vfox-busy', () => {
    if (!terminalTaskRunning.value) return;
    showTaskToast.value = true;
    showBusyHint();
  });
});

onUnmounted(() => {
  mediaQuery.removeEventListener('change', handleSystemThemeChange);
  if (offVfoxLog) {
    offVfoxLog();
    offVfoxLog = null;
  }
  if (offVfoxBusy) {
    offVfoxBusy();
    offVfoxBusy = null;
  }
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
  if (busyHintTimer) clearTimeout(busyHintTimer);
});
</script>

<template>
  <div id="layout">
    <div class="sidebar">
      <div class="logo">
        <img src="./assets/icons/icon.png" alt="logo"/>
        <h2>vfoxG</h2>
      </div>
      <nav>
        <button class="nav-btn" :class="{active: currentTab === 'sdk'}" @click="triggerSdkSidebarAction('display')">
          <span class="material-symbols-outlined">dashboard</span>
          {{ t('nav.display') }}
        </button>
        <button class="nav-btn" :class="{active: currentTab === 'sync'}" @click="switchTab('sync')">
          <span class="material-symbols-outlined">sync_alt</span>
          {{ t('nav.sync') }}
        </button>
        <button class="nav-btn" :class="{active: currentTab === 'plugin'}" @click="switchTab('plugin')">
          <span class="material-symbols-outlined">extension</span>
          {{ t('nav.market') }}
        </button>
      </nav>
      <div style="flex: 1;"></div>
      <nav style="margin-bottom: 24px;">
        <button class="nav-btn" :class="{active: currentTab === 'settings'}" @click="switchTab('settings')">
          <span class="material-symbols-outlined">settings</span>
          {{ t('nav.settings') }}
        </button>
      </nav>
    </div>
    <div class="main-shell">
      <div class="main-content">
        <Transition :name="navTransition" mode="out-in">
          <SdkManager
            v-if="currentTab === 'sdk'"
            key="sdk"
            :sidebar-action="sdkSidebarAction"
            @sidebar-action-done="clearSdkSidebarAction"
            @start-task="handleStartTask"
            @notify="handleNotify"
            @open-plugin-market="switchTab('plugin')"
            @open-sync="switchTab('sync')"
          />
          <SdkSync v-else-if="currentTab === 'sync'" key="sync" @notify="handleNotify" />
          <PluginMarket v-else-if="currentTab === 'plugin'" key="plugin" @start-task="handleStartTask" @notify="handleNotify" />
          <Settings v-else-if="currentTab === 'settings'" key="settings" @notify="handleNotify" />
        </Transition>

        <!-- Modern GUI Task Toast -->
        <Transition name="toast-slide">
          <div v-if="showTaskToast" class="task-toast" :class="`toast-${taskStatus}`">
            <div class="toast-icon">
              <div v-if="taskStatus === 'running'" class="toast-phase-icon" :class="`phase-${taskPhase}`">
                <span class="material-symbols-outlined">
                  {{ taskPhase === 'download' ? 'download' : taskPhase === 'install' ? 'inventory_2' : 'hourglass_top' }}
                </span>
              </div>
              <svg v-else-if="taskStatus === 'success'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
              <svg v-else-if="taskStatus === 'error'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
              <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
            </div>
            <div class="toast-content">
              <div class="toast-title-row">
                <div class="toast-title">{{ taskTitle }}</div>
                <div v-if="taskStatus === 'running' && taskPhase !== 'default'" class="toast-phase-pill" :class="`phase-${taskPhase}`">
                  {{ taskPhase === 'download' ? t('toast.phase.download') : t('toast.phase.install') }}
                </div>
              </div>
              <div class="toast-subtitle" :class="{'text-error': taskStatus === 'error', 'text-success': taskStatus === 'success'}">
                {{ lastLogLine }}
              </div>
              <div v-if="showToastProgress" class="toast-progress" :class="{ indeterminate: taskStatus === 'running' && !isDeterminateDownloadProgress, error: taskStatus === 'error', success: taskStatus === 'success', download: taskPhase === 'download', install: taskPhase === 'install' }">
                <div
                  class="toast-progress-fill"
                  :style="toastProgressStyle"
                ></div>
              </div>
              <div v-if="isDeterminateDownloadProgress" class="toast-progress-label">{{ taskProgress }}%</div>
              <Transition name="busy-hint">
                <div v-if="busyHintVisible" class="toast-busy-hint">{{ t('toast.please_wait') }}</div>
              </Transition>
            </div>
            <button class="toast-close" @click="closeToast">
               <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
          </div>
        </Transition>
      </div>

      <Transition name="terminal-dock-slide">
        <section v-if="showTerminal" class="terminal-dock" :aria-label="t('terminal.aria')">
          <div class="terminal-header">
            <div class="terminal-title">
              <span class="material-symbols-outlined">terminal</span>
              {{ t('terminal.title') }}
              <span class="terminal-count">{{ terminalLogs.length }}</span>
            </div>
            <div class="terminal-actions">
              <button class="terminal-icon-btn" :title="t('terminal.clear')" @click="clearTerminalLogs">
                <span class="material-symbols-outlined">delete_sweep</span>
              </button>
              <button class="terminal-icon-btn" :title="t('terminal.hide')" @click="showTerminal = false">
                <span class="material-symbols-outlined">keyboard_arrow_down</span>
              </button>
            </div>
          </div>
          <div ref="terminalBody" class="terminal-body">
            <div v-if="terminalLogs.length === 0" class="terminal-empty">{{ t('terminal.empty') }}</div>
            <div v-for="entry in terminalLogs" :key="entry.id" class="terminal-line" :class="entry.level">
              <span class="terminal-time">{{ entry.time }}</span>
              <span class="terminal-prompt">$</span>
              <span class="terminal-text">{{ entry.text }}</span>
            </div>
          </div>
        </section>
      </Transition>
    </div>
  </div>
</template>
