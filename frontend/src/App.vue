<script lang="ts" setup>
import { nextTick, provide, readonly, ref } from 'vue';
import SdkManager from './components/sdk/SdkManager.vue';
import SdkSync from './components/sync/SdkSync.vue';
import PluginMarket from './components/plugin/PluginMarket.vue';
import Settings from './components/settings/Settings.vue';
import AppSidebar from './components/app/AppSidebar.vue';
import MigrationProgressOverlay from './components/app/MigrationProgressOverlay.vue';
import TaskToast from './components/app/TaskToast.vue';
import TerminalDock from './components/app/TerminalDock.vue';
import { getNavTransition, type AppTab, type NavTransition } from './app/navigation';
import { useAppTheme } from './composables/useAppTheme';
import { useMigrationProgress } from './composables/useMigrationProgress';
import { useTaskTerminal } from './composables/useTaskTerminal';

type SdkSidebarAction = { id: number; type: 'display' };

const terminalDock = ref<InstanceType<typeof TerminalDock> | null>(null);
const scrollTerminalToBottom = () => {
  terminalDock.value?.scrollToBottom();
};

const {
  theme,
  showTerminal,
} = useAppTheme({
  onShowTerminal: () => nextTick(scrollTerminalToBottom),
});
provide('theme', theme);
provide('showTerminal', showTerminal);

const {
  showTaskToast,
  terminalTaskRunning,
  busyHintVisible,
  taskTitle,
  taskStatus,
  lastLogLine,
  taskProgress,
  taskDownloadSpeed,
  taskPhase,
  terminalLogs,
  isDeterminateDownloadProgress,
  showToastProgress,
  toastProgressStyle,
  handleStartTask,
  handleNotify,
  runTerminalTask,
  clearTerminalLogs,
} = useTaskTerminal({ showTerminal, scrollTerminalToBottom });
provide('terminalTaskRunning', readonly(terminalTaskRunning));
provide('runTerminalTask', runTerminalTask);

const {
  migrationVisible,
  migrationProgress,
  formatRemainingTime,
} = useMigrationProgress();

const currentTab = ref<AppTab>('sdk');
const navTransition = ref<NavTransition>('slide-up');
const sdkSidebarAction = ref<SdkSidebarAction | null>(null);
let sdkSidebarActionId = 0;

const switchTab = (tab: AppTab) => {
  if (tab === currentTab.value) return;
  navTransition.value = getNavTransition(currentTab.value, tab);
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
</script>

<template>
  <div id="layout">
    <AppSidebar
      :current-tab="currentTab"
      @show-sdk-display="triggerSdkSidebarAction('display')"
      @switch-tab="switchTab"
    />
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

        <TaskToast
          :visible="showTaskToast"
          :status="taskStatus"
          :phase="taskPhase"
          :title="taskTitle"
          :message="lastLogLine"
          :show-progress="showToastProgress"
          :is-determinate-download-progress="isDeterminateDownloadProgress"
          :progress="taskProgress"
          :download-speed="taskDownloadSpeed"
          :progress-style="toastProgressStyle"
          :busy-hint-visible="busyHintVisible"
        />
      </div>

      <TerminalDock
        ref="terminalDock"
        :visible="showTerminal"
        :logs="terminalLogs"
        @clear="clearTerminalLogs"
        @hide="showTerminal = false"
      />
    </div>

    <MigrationProgressOverlay
      :visible="migrationVisible"
      :progress="migrationProgress"
      :format-remaining-time="formatRemainingTime"
    />
  </div>
</template>
