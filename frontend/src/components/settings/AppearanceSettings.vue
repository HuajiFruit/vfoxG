<script lang="ts" setup>
import { currentLang, t } from '../../i18n';

defineProps<{
  theme: string;
  showTerminal: boolean;
}>();

const emit = defineEmits(['update:theme', 'update:showTerminal']);
</script>

<template>
  <div class="settings-section">
    <h3 class="section-heading">{{ t('settings.appearance') }}</h3>
    <div class="setting-card">
      <div class="setting-info">
        <h4>{{ t('settings.theme') }}</h4>
        <p>{{ t('settings.theme.desc') }}</p>
      </div>
      <div class="setting-action">
        <div class="theme-toggle">
          <label class="theme-option" :class="{ active: theme === 'light' }">
            <input type="radio" value="light" :checked="theme === 'light'" @change="emit('update:theme', 'light')">
            <span class="material-symbols-outlined">light_mode</span>
            {{ t('settings.theme.light') }}
          </label>
          <label class="theme-option" :class="{ active: theme === 'dark' }">
            <input type="radio" value="dark" :checked="theme === 'dark'" @change="emit('update:theme', 'dark')">
            <span class="material-symbols-outlined">dark_mode</span>
            {{ t('settings.theme.dark') }}
          </label>
          <label class="theme-option" :class="{ active: theme === 'auto' }">
            <input type="radio" value="auto" :checked="theme === 'auto'" @change="emit('update:theme', 'auto')">
            <span class="material-symbols-outlined">brightness_auto</span>
            {{ t('settings.theme.auto') }}
          </label>
        </div>
      </div>
    </div>

    <div class="setting-card">
      <div class="setting-info">
        <h4>{{ t('settings.language') }}</h4>
        <p>{{ t('settings.language.desc') }}</p>
      </div>
      <div class="setting-action">
        <div class="theme-toggle">
          <label class="theme-option" :class="{ active: currentLang === 'en' }">
            <input v-model="currentLang" type="radio" value="en">
            <span class="material-symbols-outlined">language</span>
            {{ t('settings.language.en') }}
          </label>
          <label class="theme-option" :class="{ active: currentLang === 'zh' }">
            <input v-model="currentLang" type="radio" value="zh">
            <span class="material-symbols-outlined">translate</span>
            {{ t('settings.language.zh') }}
          </label>
        </div>
      </div>
    </div>

    <div class="setting-card">
      <div class="setting-info">
        <h4>{{ t('settings.terminal') }}</h4>
        <p>{{ t('settings.terminal.desc') }}</p>
      </div>
      <div class="setting-action">
        <label class="switch-control" :class="{ active: showTerminal }" :title="t('settings.terminal.show')">
          <input
            type="checkbox"
            :checked="showTerminal"
            :aria-label="t('settings.terminal.show')"
            @change="emit('update:showTerminal', ($event.target as HTMLInputElement).checked)"
          >
          <span class="switch-track">
            <span class="switch-thumb"></span>
          </span>
        </label>
      </div>
    </div>
  </div>
</template>
