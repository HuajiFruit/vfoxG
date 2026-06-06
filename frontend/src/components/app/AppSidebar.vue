<script lang="ts" setup>
import { t } from '../../i18n';
import type { AppTab } from '../../app/navigation';

defineProps<{
  currentTab: AppTab;
}>();

const emit = defineEmits<{
  (event: 'show-sdk-display'): void;
  (event: 'switch-tab', tab: AppTab): void;
}>();
</script>

<template>
  <div class="sidebar">
    <div class="logo">
      <img src="../../assets/icons/icon.png" alt="logo"/>
      <h2>vfoxG</h2>
    </div>
    <nav>
      <button class="nav-btn" :class="{active: currentTab === 'sdk'}" @click="emit('show-sdk-display')">
        <span class="material-symbols-outlined">dashboard</span>
        {{ t('nav.display') }}
      </button>
      <button class="nav-btn" :class="{active: currentTab === 'sync'}" @click="emit('switch-tab', 'sync')">
        <span class="material-symbols-outlined">sync_alt</span>
        {{ t('nav.sync') }}
      </button>
      <button class="nav-btn" :class="{active: currentTab === 'plugin'}" @click="emit('switch-tab', 'plugin')">
        <span class="material-symbols-outlined">extension</span>
        {{ t('nav.market') }}
      </button>
    </nav>
    <div style="flex: 1;"></div>
    <nav style="margin-bottom: 24px;">
      <button class="nav-btn" :class="{active: currentTab === 'settings'}" @click="emit('switch-tab', 'settings')">
        <span class="material-symbols-outlined">settings</span>
        {{ t('nav.settings') }}
      </button>
    </nav>
  </div>
</template>
