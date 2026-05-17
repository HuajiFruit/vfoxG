<script lang="ts" setup>
import { ref, onMounted, onUnmounted, nextTick, watch, provide } from 'vue';
import { EventsOn, EventsOff, WindowSetLightTheme, WindowSetDarkTheme, WindowSetSystemDefaultTheme } from '../wailsjs/runtime/runtime';
import { t } from './i18n';
import SdkManager from './components/SdkManager.vue';
import PluginMarket from './components/PluginMarket.vue';
import Settings from './components/Settings.vue';

// --- Theme Management ---
const theme = ref(localStorage.getItem('vfox-theme') || 'auto');
provide('theme', theme);

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

const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
const handleSystemThemeChange = () => {
  if (theme.value === 'auto') applyTheme();
};
// ------------------------

const currentTab = ref('sdk');
const navTransition = ref('slide-up');
const showTaskToast = ref(false);

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
const taskStatus = ref<'running' | 'success' | 'error'>('running');
const lastLogLine = ref('');
let autoCloseTimer: ReturnType<typeof setTimeout> | null = null;

const handleStartTask = (title: string) => {
  taskTitle.value = title;
  lastLogLine.value = 'Starting...';
  taskStatus.value = 'running';
  showTaskToast.value = true;
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
};

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

  EventsOn('vfox-log', (log: string) => {
    // Determine status
    if (log.startsWith('[DONE]')) {
      taskStatus.value = 'success';
      lastLogLine.value = 'Completed successfully!';
    } else if (log.startsWith('[EXIT ERROR]') || log.startsWith('[TIMEOUT]')) {
      taskStatus.value = 'error';
      lastLogLine.value = log.replace('[EXIT ERROR] ', '').replace('[TIMEOUT] ', '');
    } else {
      lastLogLine.value = log;
    }

    if (taskStatus.value !== 'running') {
      if (autoCloseTimer) clearTimeout(autoCloseTimer);
      autoCloseTimer = setTimeout(() => {
        showTaskToast.value = false;
      }, 2500); // Wait a bit longer to show success/error state before closing
    }
  });
});

onUnmounted(() => {
  mediaQuery.removeEventListener('change', handleSystemThemeChange);
  EventsOff('vfox-log');
  if (autoCloseTimer) clearTimeout(autoCloseTimer);
});
</script>

<template>
  <div id="layout">
    <div class="sidebar">
      <div class="logo">
        <img src="./assets/icons/icon.png" alt="logo"/>
        <h2>vfoxN</h2>
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
    <div class="main-content">
      <Transition :name="navTransition" mode="out-in">
        <SdkManager v-if="currentTab === 'sdk'" key="sdk" @start-task="handleStartTask" />
        <PluginMarket v-else-if="currentTab === 'plugin'" key="plugin" @start-task="handleStartTask" />
        <Settings v-else-if="currentTab === 'settings'" key="settings" />
      </Transition>

      <!-- Modern GUI Task Toast -->
      <Transition name="toast-slide">
        <div v-if="showTaskToast" class="task-toast">
          <div class="toast-icon">
            <div v-if="taskStatus === 'running'" class="spinner small-spinner"></div>
            <svg v-else-if="taskStatus === 'success'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
            <svg v-else-if="taskStatus === 'error'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#ef4444" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
          </div>
          <div class="toast-content">
            <div class="toast-title">{{ taskTitle }}</div>
            <div class="toast-subtitle" :class="{'text-error': taskStatus === 'error', 'text-success': taskStatus === 'success'}">
              {{ lastLogLine }}
            </div>
          </div>
          <button class="toast-close" @click="closeToast">
             <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
          </button>
        </div>
      </Transition>
    </div>
  </div>
</template>
