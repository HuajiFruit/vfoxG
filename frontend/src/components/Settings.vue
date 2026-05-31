<script lang="ts" setup>
import { computed, inject, Ref, ref, onMounted, watch } from 'vue';
import {
  GetDownloadPathInfo,
  SetDownloadPath,
  SetDownloadPathWithMigration,
  ResetDownloadPath,
  ResetDownloadPathWithMigration,
  SelectDownloadPath,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import { t, currentLang } from '../i18n';

const theme = inject<Ref<string>>('theme');
const showTerminal = inject<Ref<boolean>>('showTerminal');
const downloadPathInfo = ref<main.DownloadPathInfo | null>(null);
const downloadPathInput = ref('');
const loadingDownloadPath = ref(true);
const savingDownloadPath = ref(false);
const selectingDownloadPath = ref(false);
const resettingDownloadPath = ref(false);

type PendingMigrationAction = {
  type: 'save' | 'reset';
  targetPath: string;
};

const pendingMigration = ref<PendingMigrationAction | null>(null);

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

const syncDownloadPathInfo = (info: main.DownloadPathInfo) => {
  downloadPathInfo.value = info;
  downloadPathInput.value = info.path;
};

const shouldAskMigration = (targetPath: string) => {
  const currentPath = downloadPathInfo.value?.path?.trim();
  return Boolean(
    downloadPathInfo.value?.hasMigratableData &&
    currentPath &&
    targetPath.trim() &&
    currentPath !== targetPath.trim()
  );
};

const isDownloadPathBusy = computed(() => savingDownloadPath.value || resettingDownloadPath.value);

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
  const targetPath = downloadPathInput.value.trim();
  if (shouldAskMigration(targetPath)) {
    pendingMigration.value = { type: 'save', targetPath };
    return;
  }

  pendingMigration.value = null;
  savingDownloadPath.value = true;
  try {
    const info = await SetDownloadPath(targetPath);
    syncDownloadPathInfo(info);
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
  const targetPath = downloadPathInfo.value?.defaultPath || '';
  if (shouldAskMigration(targetPath)) {
    pendingMigration.value = { type: 'reset', targetPath };
    return;
  }

  pendingMigration.value = null;
  resettingDownloadPath.value = true;
  try {
    const info = await ResetDownloadPath();
    syncDownloadPathInfo(info);
    notifySuccess(t('settings.download.path.reset_success'));
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.download.path.error')));
  } finally {
    resettingDownloadPath.value = false;
  }
};

const confirmPendingMigration = async () => {
  const action = pendingMigration.value;
  if (!action || isDownloadPathBusy.value) return;

  if (action.type === 'save') {
    savingDownloadPath.value = true;
    try {
      const info = await SetDownloadPathWithMigration(action.targetPath, true);
      pendingMigration.value = null;
      syncDownloadPathInfo(info);
      notifySuccess(t('settings.download.path.migrate_success'));
    } catch (err) {
      notifyError(getErrorMessage(err, t('settings.download.path.error')));
    } finally {
      savingDownloadPath.value = false;
    }
    return;
  }

  resettingDownloadPath.value = true;
  try {
    const info = await ResetDownloadPathWithMigration(true);
    pendingMigration.value = null;
    syncDownloadPathInfo(info);
    notifySuccess(t('settings.download.path.migrate_success'));
  } catch (err) {
    notifyError(getErrorMessage(err, t('settings.download.path.error')));
  } finally {
    resettingDownloadPath.value = false;
  }
};

const cancelPendingMigration = () => {
  if (isDownloadPathBusy.value) return;
  pendingMigration.value = null;
};

watch(downloadPathInput, (value) => {
  if (pendingMigration.value?.type === 'save' && value.trim() !== pendingMigration.value.targetPath) {
    pendingMigration.value = null;
  }
});

onMounted(() => {
  loadDownloadPath();
});

</script>

<template>
  <div class="settings-view view-container">
    <h2>{{ t('settings.title') }}</h2>

    <div v-if="pendingMigration" class="migration-confirm-panel">
      <div class="migration-confirm-icon">
        <span class="material-symbols-outlined">sync_alt</span>
      </div>
      <div class="migration-confirm-copy">
        <h3>{{ t('settings.download.path.migrate_title') }}</h3>
        <p>{{ t('settings.download.path.migrate_confirm') }}</p>
        <div class="migration-confirm-path">
          <span>{{ t('settings.download.path.migrate_target') }}</span>
          <code>{{ pendingMigration.targetPath }}</code>
        </div>
      </div>
      <div class="migration-confirm-actions">
        <button class="btn outlined" :disabled="isDownloadPathBusy" @click="cancelPendingMigration">
          {{ t('common.cancel') }}
        </button>
        <button class="btn primary" :disabled="isDownloadPathBusy" @click="confirmPendingMigration">
          <div v-if="isDownloadPathBusy" class="spinner small-spinner"></div>
          <template v-else>{{ t('common.confirm') }}</template>
        </button>
      </div>
    </div>
    
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
          <label class="switch-control" :class="{ active: terminalVisible }" :title="t('settings.terminal.show')">
            <input type="checkbox" v-model="terminalVisible" :aria-label="t('settings.terminal.show')">
            <span class="switch-track">
              <span class="switch-thumb"></span>
            </span>
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

.migration-confirm-panel {
  margin-top: 16px;
  padding: 16px;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  border: 1px solid rgba(255, 188, 77, 0.48);
  border-radius: var(--md-shape-medium);
  background:
    linear-gradient(135deg, rgba(255, 188, 77, 0.15), rgba(115, 214, 208, 0.06)),
    var(--md-surface-container-low);
}

.migration-confirm-icon {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--md-shape-small);
  background: rgba(255, 188, 77, 0.14);
  color: var(--md-on-surface);
}

.migration-confirm-icon span {
  font-size: 24px;
}

.migration-confirm-copy {
  min-width: 0;
}

.migration-confirm-copy h3 {
  margin: 0 0 4px;
  color: var(--md-on-surface);
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0;
}

.migration-confirm-copy p {
  margin: 0;
  color: var(--md-on-surface-variant);
  font-size: 13px;
  line-height: 1.5;
}

.migration-confirm-path {
  margin-top: 8px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 8px;
}

.migration-confirm-path span {
  color: var(--md-on-surface-variant);
  font-size: 11px;
  font-weight: 650;
  text-transform: uppercase;
}

.migration-confirm-path code {
  min-width: 0;
  overflow: hidden;
  color: var(--md-on-surface);
  font-family: "Cascadia Mono", "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.migration-confirm-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
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
  width: 52px;
  height: 32px;
  cursor: pointer;
  user-select: none;
}

.switch-control input {
  display: none;
}

.switch-track {
  width: 100%;
  height: 100%;
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
  border-radius: 50%;
  background: var(--md-outline);
  transform: translateX(0);
  transition: transform 180ms cubic-bezier(0.2, 0, 0, 1),
              background-color 180ms cubic-bezier(0.2, 0, 0, 1),
              box-shadow 180ms cubic-bezier(0.2, 0, 0, 1);
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
  transform: translateX(20px);
  box-shadow: 0 6px 12px rgba(0, 0, 0, 0.18);
}

@media (max-width: 980px) {
  .migration-confirm-panel {
    grid-template-columns: 42px minmax(0, 1fr);
    align-items: flex-start;
  }

  .migration-confirm-actions {
    width: 100%;
    grid-column: 1 / -1;
    justify-content: flex-start;
  }

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
