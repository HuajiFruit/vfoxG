<script lang="ts" setup>
import { inject, Ref, ref, onMounted } from 'vue';
import { CheckVfoxInPath, AddVfoxToPath, RemoveVfoxFromPath } from '../../wailsjs/go/main/App';
import { t, currentLang } from '../i18n';

const theme = inject<Ref<string>>('theme');
const isInPath = ref(false);
const addingToPath = ref(false);
const removingFromPath = ref(false);
const checkingPath = ref(true);

const checkPath = async () => {
  checkingPath.value = true;
  try {
    isInPath.value = await CheckVfoxInPath();
  } catch (err) {
    console.error(err);
  } finally {
    checkingPath.value = false;
  }
};

const addToPath = async () => {
  addingToPath.value = true;
  try {
    await AddVfoxToPath();
    await checkPath();
  } catch (err) {
    console.error(err);
  } finally {
    addingToPath.value = false;
  }
};

const removeFromPath = async () => {
  removingFromPath.value = true;
  try {
    await RemoveVfoxFromPath();
    await checkPath();
  } catch (err) {
    console.error(err);
  } finally {
    removingFromPath.value = false;
  }
};

onMounted(() => {
  checkPath();
});

</script>

<template>
  <div class="settings-view view-container">
    <h2>{{ t('settings.title') }}</h2>
    
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
              <input type="radio" value="light" v-model="theme">
              <span class="material-symbols-outlined">light_mode</span>
              {{ t('settings.theme.light') }}
            </label>
            <label class="theme-option" :class="{ active: theme === 'dark' }">
              <input type="radio" value="dark" v-model="theme">
              <span class="material-symbols-outlined">dark_mode</span>
              {{ t('settings.theme.dark') }}
            </label>
            <label class="theme-option" :class="{ active: theme === 'auto' }">
              <input type="radio" value="auto" v-model="theme">
              <span class="material-symbols-outlined">brightness_auto</span>
              {{ t('settings.theme.auto') }}
            </label>
          </div>
        </div>
      </div>
      
      <div class="setting-card" style="margin-top: 16px;">
        <div class="setting-info">
          <h4>{{ t('settings.language') }}</h4>
          <p>{{ t('settings.language.desc') }}</p>
        </div>
        <div class="setting-action">
          <div class="theme-toggle">
            <label class="theme-option" :class="{ active: currentLang === 'en' }">
              <input type="radio" value="en" v-model="currentLang">
              <span class="material-symbols-outlined">language</span>
              {{ t('settings.language.en') }}
            </label>
            <label class="theme-option" :class="{ active: currentLang === 'zh' }">
              <input type="radio" value="zh" v-model="currentLang">
              <span class="material-symbols-outlined">translate</span>
              {{ t('settings.language.zh') }}
            </label>
          </div>
        </div>
      </div>
    </div>
    
    <div class="settings-section">
      <h3 class="section-heading">{{ t('settings.system') }}</h3>
      <div class="setting-card">
        <div class="setting-info">
          <h4>{{ t('settings.system.path') }}</h4>
          <p>{{ t('settings.system.path.desc') }}</p>
        </div>
        <div class="setting-action">
          <button 
            v-if="checkingPath"
            class="btn" 
            disabled
            style="min-width: 160px; display: flex; justify-content: center; align-items: center; background: transparent;"
          >
            <div class="spinner small-spinner" style="width: 20px; height: 20px; border-width: 2px; border-color: var(--md-outline) transparent var(--md-outline) transparent;"></div>
          </button>
          <button 
            v-else-if="!isInPath"
            class="btn primary" 
            :disabled="addingToPath"
            @click="addToPath"
            style="min-width: 140px; display: flex; justify-content: center; align-items: center;"
          >
            <div v-if="addingToPath" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px;"></div>
            <template v-else>
              {{ t('settings.system.path.add') }}
            </template>
          </button>
          
          <button 
            v-else
            class="btn outlined danger" 
            :disabled="removingFromPath"
            @click="removeFromPath"
            style="min-width: 160px;"
          >
            <div v-if="removingFromPath" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px; border-color: var(--md-error) transparent var(--md-error) transparent;"></div>
            <template v-else>
              <span class="material-symbols-outlined" style="font-size: 18px; margin-right: 6px;">delete</span>
              {{ t('settings.system.path.remove') }}
            </template>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-view {
  display: flex;
  flex-direction: column;
}

.settings-section {
  margin-top: 24px;
}

.setting-card {
  background-color: var(--md-surface-container);
  border: 1px solid var(--md-outline-variant);
  border-radius: var(--md-shape-medium);
  padding: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.setting-info h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
  color: var(--md-on-surface);
}

.setting-info p {
  margin: 0;
  font-size: 14px;
  color: var(--md-on-surface-variant);
}

.theme-toggle {
  display: flex;
  background-color: var(--md-surface-container-highest);
  border-radius: var(--md-shape-full);
  padding: 4px;
  gap: 4px;
}

.theme-option {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: var(--md-shape-full);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--md-on-surface-variant);
  transition: all 0.2s ease;
}

.theme-option input {
  display: none;
}

.theme-option span {
  font-size: 18px;
}

.theme-option:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.theme-option.active {
  background-color: var(--md-primary);
  color: var(--md-on-primary);
}

[data-theme="light"] .theme-option:hover {
  background-color: rgba(0, 0, 0, 0.05);
}
</style>
