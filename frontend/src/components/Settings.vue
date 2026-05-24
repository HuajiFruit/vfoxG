<script lang="ts" setup>
import { computed, inject, Ref, ref, onMounted } from 'vue';
import {
  CheckVfoxInPath,
  AddVfoxToPath,
  RemoveVfoxFromPath,
  GetPlatformInfo,
  GetDownloadPathInfo,
  SetDownloadPath,
  ResetDownloadPath,
  SelectDownloadPath,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import { t, currentLang } from '../i18n';

const theme = inject<Ref<string>>('theme');
const showTerminal = inject<Ref<boolean>>('showTerminal');
const isInPath = ref(false);
const addingToPath = ref(false);
const removingFromPath = ref(false);
const checkingPath = ref(true);
const platformInfo = ref<main.PlatformInfo | null>(null);
const downloadPathInfo = ref<main.DownloadPathInfo | null>(null);
const downloadPathInput = ref('');
const loadingDownloadPath = ref(true);
const savingDownloadPath = ref(false);
const selectingDownloadPath = ref(false);
const resettingDownloadPath = ref(false);

const emit = defineEmits(['notify']);

const getErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'string' && err.trim()) return err;
  return fallback;
};

const notifyError = (message: string, title = t('common.error')) => {
  emit('notify', { type: 'error', title, message });
};

const notifySuccess = (message: string, title = t('common.success')) => {
  emit('notify', { type: 'success', title, message });
};

const terminalVisible = computed({
  get: () => showTerminal?.value ?? false,
  set: (value: boolean) => {
    if (showTerminal) {
      showTerminal.value = value;
    }
  },
});

const platformRestartHint = computed(() => {
  const os = platformInfo.value?.os || 'default';
  return t(`platform.restart.${os}`) === `platform.restart.${os}`
    ? t('platform.restart.default')
    : t(`platform.restart.${os}`);
});

const pathDescription = computed(() => {
  if (!platformInfo.value) return t('settings.system.path.desc');
  return t('settings.system.path.desc.platform', {
    target: platformInfo.value.vfoxPathTarget,
    restart: platformRestartHint.value,
  });
});

const syncDownloadPathInfo = (info: main.DownloadPathInfo) => {
  downloadPathInfo.value = info;
  downloadPathInput.value = info.path;
};

const loadPlatformInfo = async () => {
  try {
    platformInfo.value = await GetPlatformInfo();
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.platform.load_error')));
  }
};

const loadDownloadPath = async () => {
  loadingDownloadPath.value = true;
  try {
    syncDownloadPathInfo(await GetDownloadPathInfo());
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.download.path.load_error')));
  } finally {
    loadingDownloadPath.value = false;
  }
};

const saveDownloadPath = async () => {
  savingDownloadPath.value = true;
  try {
    syncDownloadPathInfo(await SetDownloadPath(downloadPathInput.value));
    await loadPlatformInfo();
    notifySuccess(t('settings.download.path.success'));
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.download.path.error')));
  } finally {
    savingDownloadPath.value = false;
  }
};

const chooseDownloadPath = async () => {
  selectingDownloadPath.value = true;
  try {
    const selected = await SelectDownloadPath();
    if (selected && selected.trim()) {
      downloadPathInput.value = selected;
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.download.path.select_error')));
  } finally {
    selectingDownloadPath.value = false;
  }
};

const resetDownloadPath = async () => {
  resettingDownloadPath.value = true;
  try {
    syncDownloadPathInfo(await ResetDownloadPath());
    await loadPlatformInfo();
    notifySuccess(t('settings.download.path.reset_success'));
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.download.path.error')));
  } finally {
    resettingDownloadPath.value = false;
  }
};

const checkPath = async (notifyOnError = true) => {
  checkingPath.value = true;
  try {
    isInPath.value = await CheckVfoxInPath();
    return true;
  } catch (err) {
    if (notifyOnError) {
      notifyError(getErrorMessage(err, t('settings.path.check_error')));
    }
    return false;
  } finally {
    checkingPath.value = false;
  }
};

const addToPath = async () => {
  addingToPath.value = true;
  try {
    await AddVfoxToPath();
    const verified = await checkPath(false);
    if (!verified) {
      notifyError(t('settings.path.add_verify_error'));
      return;
    }
    notifySuccess(t('settings.path.add_success', { restart: platformRestartHint.value }));
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.path.add_error')));
  } finally {
    addingToPath.value = false;
  }
};

const removeFromPath = async () => {
  removingFromPath.value = true;
  try {
    await RemoveVfoxFromPath();
    const verified = await checkPath(false);
    if (!verified) {
      notifyError(t('settings.path.remove_verify_error'));
      return;
    }
    notifySuccess(t('settings.path.remove_success', { restart: platformRestartHint.value }));
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.path.remove_error')));
  } finally {
    removingFromPath.value = false;
  }
};

onMounted(() => {
  loadPlatformInfo();
  loadDownloadPath();
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

      <div class="setting-card" style="margin-top: 16px;">
        <div class="setting-info">
          <h4>{{ t('settings.terminal') }}</h4>
          <p>{{ t('settings.terminal.desc') }}</p>
        </div>
        <div class="setting-action">
          <label class="switch-control" :class="{ active: terminalVisible }">
            <input type="checkbox" v-model="terminalVisible">
            <span class="switch-track">
              <span class="switch-thumb">
                <span class="material-symbols-outlined">{{ terminalVisible ? 'terminal' : 'terminal_off' }}</span>
              </span>
            </span>
            <span class="switch-label">{{ t('settings.terminal.show') }}</span>
          </label>
        </div>
      </div>
    </div>
    
    <div class="settings-section">
      <h3 class="section-heading">{{ t('settings.system') }}</h3>
      <div class="setting-card setting-card-column">
        <div class="setting-card-header">
          <div class="setting-info">
            <h4>{{ t('settings.download.path') }}</h4>
            <p>{{ t('settings.download.path.desc') }}</p>
          </div>
          <span class="path-state-pill" :class="{ default: downloadPathInfo?.isDefault }">
            {{ downloadPathInfo?.isDefault ? t('settings.download.path.default_state') : t('settings.download.path.custom_state') }}
          </span>
        </div>
        <div class="path-setting">
          <div class="path-input-row">
            <input
              v-model="downloadPathInput"
              class="path-input"
              :placeholder="downloadPathInfo?.defaultPath || t('settings.download.path.placeholder')"
              :disabled="loadingDownloadPath || savingDownloadPath || resettingDownloadPath"
              @keyup.enter="saveDownloadPath"
            >
            <button
              class="path-icon-btn"
              :title="t('settings.download.path.browse')"
              :disabled="selectingDownloadPath || savingDownloadPath || resettingDownloadPath"
              @click="chooseDownloadPath"
            >
              <span v-if="!selectingDownloadPath" class="material-symbols-outlined">folder_open</span>
              <div v-else class="spinner small-spinner"></div>
            </button>
            <button
              class="btn primary"
              :disabled="savingDownloadPath || resettingDownloadPath || !downloadPathInput.trim()"
              @click="saveDownloadPath"
            >
              <div v-if="savingDownloadPath" class="spinner small-spinner"></div>
              <template v-else>{{ t('settings.download.path.save') }}</template>
            </button>
            <button
              class="btn outlined"
              :disabled="resettingDownloadPath || savingDownloadPath || downloadPathInfo?.isDefault"
              @click="resetDownloadPath"
            >
              <div v-if="resettingDownloadPath" class="spinner small-spinner"></div>
              <template v-else>{{ t('settings.download.path.reset') }}</template>
            </button>
          </div>
          <div class="path-meta">
            <div class="path-meta-row">
              <span>{{ t('settings.download.path.current') }}</span>
              <code>{{ downloadPathInfo?.path || '-' }}</code>
            </div>
            <div class="path-meta-row">
              <span>{{ t('settings.download.path.default') }}</span>
              <code>{{ downloadPathInfo?.defaultPath || '-' }}</code>
            </div>
          </div>
        </div>
      </div>

      <div class="setting-card">
        <div class="setting-info">
          <h4>{{ t('settings.system.path') }}</h4>
          <p>{{ pathDescription }}</p>
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
  background:
    linear-gradient(180deg, rgba(115, 214, 208, 0.04), transparent 72%),
    var(--md-surface-container-low);
  border: 1px solid var(--panel-border, var(--md-outline-variant));
  border-radius: var(--md-shape-medium);
  padding: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  box-shadow: none;
  transition: border-color 180ms cubic-bezier(0.2, 0, 0, 1),
              box-shadow 180ms cubic-bezier(0.2, 0, 0, 1),
              transform 180ms cubic-bezier(0.2, 0, 0, 1);
}

.setting-card:hover {
  border-color: var(--accent-cyan, var(--md-primary));
  box-shadow: var(--panel-shadow-soft, var(--md-elevation-2));
  transform: translateY(-1px);
}

.setting-card + .setting-card {
  margin-top: 16px;
}

.setting-card-column {
  align-items: stretch;
  flex-direction: column;
  gap: 16px;
}

.setting-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.setting-info {
  min-width: 0;
  max-width: 680px;
}

.setting-info h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 650;
  color: var(--md-on-surface);
  letter-spacing: 0;
}

.setting-info p {
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--md-on-surface-variant);
}

.setting-action {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.path-setting {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.path-input-row {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) 44px auto auto;
  align-items: center;
  gap: 10px;
}

.path-input {
  width: 100%;
  min-width: 0;
  height: 42px;
  padding: 0 14px;
  border-radius: var(--md-shape-small);
  border: 1px solid var(--panel-border, var(--md-outline-variant));
  background: var(--md-surface-container-lowest);
  color: var(--md-on-surface);
  font: inherit;
  font-size: 13px;
  outline: none;
  transition: border-color 180ms cubic-bezier(0.2, 0, 0, 1),
              box-shadow 180ms cubic-bezier(0.2, 0, 0, 1);
}

.path-input:focus {
  border-color: var(--accent-cyan, var(--md-primary));
  box-shadow: 0 0 0 3px rgba(115, 214, 208, 0.14);
}

.path-input:disabled {
  opacity: 0.68;
}

.path-icon-btn {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--md-shape-small);
  border: 1px solid var(--panel-border, var(--md-outline-variant));
  background: var(--md-surface-container-lowest);
  color: var(--md-on-surface-variant);
  cursor: pointer;
  transition: border-color 180ms cubic-bezier(0.2, 0, 0, 1),
              color 180ms cubic-bezier(0.2, 0, 0, 1),
              transform 180ms cubic-bezier(0.2, 0, 0, 1);
}

.path-icon-btn:hover:not(:disabled) {
  border-color: var(--accent-cyan, var(--md-primary));
  color: var(--md-on-surface);
  transform: translateY(-1px);
}

.path-icon-btn:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.path-meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.path-meta-row {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--panel-border, var(--md-outline-variant));
  border-radius: var(--md-shape-small);
  background: rgba(115, 214, 208, 0.05);
}

.path-meta-row span {
  display: block;
  margin-bottom: 4px;
  font-size: 11px;
  font-weight: 650;
  color: var(--md-on-surface-variant);
  text-transform: uppercase;
}

.path-meta-row code {
  display: block;
  overflow: hidden;
  color: var(--md-on-surface);
  font-family: "Cascadia Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path-state-pill {
  flex: 0 0 auto;
  min-height: 26px;
  display: inline-flex;
  align-items: center;
  border-radius: var(--md-shape-full);
  padding: 0 10px;
  background: var(--md-surface-container-lowest);
  border: 1px solid var(--panel-border, var(--md-outline-variant));
  color: var(--md-on-surface-variant);
  font-size: 12px;
  font-weight: 650;
  white-space: nowrap;
}

.path-state-pill.default {
  background: rgba(115, 214, 208, 0.12);
  border-color: var(--accent-cyan, var(--md-primary));
  color: var(--md-on-surface);
}

.theme-toggle {
  display: flex;
  background-color: var(--md-surface-container-lowest);
  border: 1px solid var(--panel-border, var(--md-outline-variant));
  border-radius: var(--md-shape-medium);
  padding: 4px;
  gap: 4px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.theme-option {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 7px 13px;
  border-radius: var(--md-shape-small);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--md-on-surface-variant);
  white-space: nowrap;
  transition: background-color 180ms cubic-bezier(0.2, 0, 0, 1),
              color 180ms cubic-bezier(0.2, 0, 0, 1),
              box-shadow 180ms cubic-bezier(0.2, 0, 0, 1);
}

.theme-option input {
  display: none;
}

.theme-option span {
  font-size: 18px;
}

.theme-option:hover {
  background-color: rgba(115, 214, 208, 0.08);
  color: var(--md-on-surface);
}

.theme-option.active {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.1), transparent),
    var(--md-primary);
  color: var(--md-on-primary);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.16);
}

[data-theme="light"] .theme-option:hover {
  background-color: rgba(34, 157, 150, 0.08);
}

.switch-control {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  color: var(--md-on-surface-variant);
  font-size: 14px;
  font-weight: 500;
  user-select: none;
  padding: 4px 0;
  transition: color 180ms cubic-bezier(0.2, 0, 0, 1);
}

.switch-control input {
  display: none;
}

.switch-track {
  width: 52px;
  height: 32px;
  padding: 3px;
  display: inline-flex;
  align-items: center;
  border-radius: var(--md-shape-full);
  background: var(--md-surface-container-lowest);
  border: 1px solid var(--panel-border, var(--md-outline));
  transition: background-color 180ms cubic-bezier(0.2, 0, 0, 1),
              border-color 180ms cubic-bezier(0.2, 0, 0, 1),
              box-shadow 180ms cubic-bezier(0.2, 0, 0, 1);
}

.switch-thumb {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--md-outline);
  color: var(--md-surface-container-highest);
  transform: translateX(0);
  transition: transform 180ms cubic-bezier(0.2, 0, 0, 1),
              background-color 180ms cubic-bezier(0.2, 0, 0, 1),
              color 180ms cubic-bezier(0.2, 0, 0, 1),
              box-shadow 180ms cubic-bezier(0.2, 0, 0, 1);
}

.switch-thumb .material-symbols-outlined {
  font-size: 15px;
}

.switch-control.active {
  color: var(--md-on-surface);
}

.switch-control.active .switch-track {
  background:
    linear-gradient(180deg, rgba(115, 214, 208, 0.18), transparent),
    var(--md-primary-container);
  border-color: var(--accent-cyan, var(--md-primary));
  box-shadow: 0 0 0 3px rgba(115, 214, 208, 0.12);
}

.switch-control.active .switch-thumb {
  background: var(--accent-cyan, var(--md-primary));
  color: #0d1718;
  transform: translateX(20px);
  box-shadow: 0 6px 12px rgba(0, 0, 0, 0.18);
}

.switch-label {
  white-space: nowrap;
}

@media (max-width: 980px) {
  .setting-card {
    align-items: flex-start;
    flex-direction: column;
  }

  .setting-action {
    width: 100%;
    justify-content: flex-start;
  }

  .setting-card-header {
    flex-direction: column;
  }

  .path-input-row {
    grid-template-columns: minmax(0, 1fr) 44px;
  }

  .path-input-row .btn {
    justify-content: center;
    grid-column: 1 / -1;
  }

  .path-meta {
    grid-template-columns: 1fr;
  }

  .theme-toggle {
    max-width: 100%;
    flex-wrap: wrap;
  }
}
</style>
