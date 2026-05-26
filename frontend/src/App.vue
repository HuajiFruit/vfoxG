<script lang="ts" setup>
import { ref, onMounted, onUnmounted, nextTick, watch, provide } from 'vue';
import { EventsOn, WindowSetLightTheme, WindowSetDarkTheme, WindowSetSystemDefaultTheme } from '../wailsjs/runtime/runtime';
import { t } from './i18n';
import SdkManager from './components/SdkManager.vue';
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
type ToastStatus = 'running' | 'success' | 'error' | 'info';
type NotifyPayload = string | {
  title?: string;
  message: string;
  type?: Exclude<ToastStatus, 'running'>;
  durationMs?: number;
};

const switchTab = (tab: string) => {
  if (tab === currentTab.value) return;
  if (currentTab.value === 'sdk' && (tab === 'plugin' || tab === 'settings')) {
    navTransition.value = 'slide-up';
  } else if (currentTab.value === 'plugin' && tab === 'settings') {
    navTransition.value = 'slide-up';
  } else {
    navTransition.value = 'slide-down';
  }
  currentTab.value = tab;
};
const taskTitle = ref('');
const taskStatus = ref<ToastStatus>('running');
const lastLogLine = ref('');
const taskProgress = ref(0);
const hasTaskProgress = ref(false);
const taskHadError = ref(false);
let autoCloseTimer: ReturnType<typeof setTimeout> | null = null;
let offVfoxLog: (() => void) | null = null;
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
  if (
    log.startsWith('[EXIT ERROR]') ||
    log.startsWith('[TIMEOUT]') ||
    log.startsWith('[APP ERROR]') ||
    log.startsWith('[STDOUT READ ERROR]') ||
    log.startsWith('[STDERR READ ERROR]')
  ) {
    return 'error';
  }
  if (log.startsWith('[DONE]')) return 'success';
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

const handleStartTask = (title: string) => {
  taskTitle.value = title;
  lastLogLine.value = t('toast.starting');
  taskStatus.value = 'running';
  taskProgress.value = 0;
  hasTaskProgress.value = false;
  taskHadError.value = false;
  showTaskToast.value = true;
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
};

const scheduleToastClose = (durationMs: number) => {
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
  autoCloseTimer = setTimeout(() => {
    showTaskToast.value = false;
  }, durationMs);
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

    const parsedProgress = extractProgressPercent(log);
    if (parsedProgress !== null) {
      taskProgress.value = parsedProgress;
      hasTaskProgress.value = true;
    }

    showTaskToast.value = true;

    // Determine status
    if (log.startsWith('[DONE]')) {
      if (taskHadError.value) {
        scheduleToastClose(5000);
        return;
      }
      taskStatus.value = 'success';
      lastLogLine.value = t('toast.completed');
      taskProgress.value = 100;
      hasTaskProgress.value = true;
    } else if (
      log.startsWith('[EXIT ERROR]') ||
      log.startsWith('[TIMEOUT]') ||
      log.startsWith('[APP ERROR]') ||
      log.startsWith('[STDOUT READ ERROR]') ||
      log.startsWith('[STDERR READ ERROR]')
    ) {
      taskHadError.value = true;
      taskStatus.value = 'error';
      lastLogLine.value = formatTaskError(log);
    } else if (!taskHadError.value) {
      lastLogLine.value = log;
    }

    if (taskStatus.value !== 'running') {
      scheduleToastClose(taskStatus.value === 'error' ? 5000 : 2500); // Wait a bit longer to show success/error state before closing
    }
  });
});

onUnmounted(() => {
  mediaQuery.removeEventListener('change', handleSystemThemeChange);
  if (offVfoxLog) {
    offVfoxLog();
    offVfoxLog = null;
  }
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
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
        <button class="nav-btn" :class="{active: currentTab === 'sdk'}" @click="switchTab('sdk')">
          <span class="material-symbols-outlined">box</span>
          {{ t('nav.installed') }}
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
          <SdkManager v-if="currentTab === 'sdk'" key="sdk" @start-task="handleStartTask" @notify="handleNotify" />
          <PluginMarket v-else-if="currentTab === 'plugin'" key="plugin" @start-task="handleStartTask" @notify="handleNotify" />
          <Settings v-else-if="currentTab === 'settings'" key="settings" @notify="handleNotify" />
        </Transition>

        <!-- Modern GUI Task Toast -->
        <Transition name="toast-slide">
          <div v-if="showTaskToast" class="task-toast" :class="`toast-${taskStatus}`">
            <div class="toast-icon">
              <div v-if="taskStatus === 'running'" class="spinner small-spinner"></div>
              <svg v-else-if="taskStatus === 'success'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
              <svg v-else-if="taskStatus === 'error'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
              <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--md-primary)" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
            </div>
            <div class="toast-content">
              <div class="toast-title">{{ taskTitle }}</div>
              <div class="toast-subtitle" :class="{'text-error': taskStatus === 'error', 'text-success': taskStatus === 'success'}">
                {{ lastLogLine }}
              </div>
              <div v-if="taskStatus === 'running' || hasTaskProgress" class="toast-progress" :class="{ indeterminate: taskStatus === 'running' && !hasTaskProgress, error: taskStatus === 'error', success: taskStatus === 'success' }">
                <div
                  class="toast-progress-fill"
                  :style="hasTaskProgress ? { width: `${taskProgress}%` } : undefined"
                ></div>
              </div>
              <div v-if="hasTaskProgress && taskStatus === 'running'" class="toast-progress-label">{{ taskProgress }}%</div>
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
