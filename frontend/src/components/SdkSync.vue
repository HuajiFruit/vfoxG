<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import {
  ExportCurrentEnvironmentSdks,
  ImportSdkEnvironmentFromTxt,
  PreviewCurrentEnvironmentSdks,
} from '../../wailsjs/go/main/App';
import { t } from '../i18n';

const emit = defineEmits(['notify']);

const previewText = ref('');
const loadingPreview = ref(true);
const exporting = ref(false);
const importing = ref(false);
const lastUpdated = ref('');

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

const loadPreview = async () => {
  loadingPreview.value = true;
  try {
    previewText.value = await PreviewCurrentEnvironmentSdks();
    lastUpdated.value = new Date().toLocaleString();
  } catch (err) {
    previewText.value = '';
    notifyError(getErrorMessage(err, t('sync.preview_error')));
  } finally {
    loadingPreview.value = false;
  }
};

const handleExport = async () => {
  if (exporting.value) return;
  exporting.value = true;
  try {
    const path = await ExportCurrentEnvironmentSdks();
    if (path) {
      notifySuccess(t('sdk.export.success', { path }));
      await loadPreview();
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.export.error')));
  } finally {
    exporting.value = false;
  }
};

const handleImport = async () => {
  if (importing.value) return;
  importing.value = true;
  try {
    const result = await ImportSdkEnvironmentFromTxt();
    if (result?.path) {
      const message = t('sdk.import.success', {
        imported: result.importedCustomSdks ?? 0,
        skipped: result.skippedCustomSdks ?? 0,
        vfox: result.vfoxSdksFound ?? 0,
        installed: result.installedVfoxSdks ?? 0,
        vfoxSkipped: result.skippedVfoxSdks ?? 0,
      });
      const warningText = result.warnings?.length ? ` ${result.warnings.join(' ')}` : '';
      notifySuccess(`${message}${warningText}`);
      await loadPreview();
    }
  } catch (err) {
    notifyError(getErrorMessage(err, t('sdk.import.error')));
  } finally {
    importing.value = false;
  }
};

onMounted(loadPreview);
</script>

<template>
  <div class="sync-view view-container">
    <div class="sync-header">
      <div>
        <h2>{{ t('sync.title') }}</h2>
        <div v-if="lastUpdated" class="sync-updated">{{ t('sync.updated_at', { time: lastUpdated }) }}</div>
      </div>
      <div class="sync-actions">
        <button class="btn tonal" :disabled="exporting || importing" @click="handleExport">
          <span v-if="exporting" class="spinner small-spinner"></span>
          <span v-else class="material-symbols-outlined">download</span>
          {{ t('sdk.export') }}
        </button>
        <button class="btn primary" :disabled="importing || exporting" @click="handleImport">
          <span v-if="importing" class="spinner small-spinner"></span>
          <span v-else class="material-symbols-outlined">upload_file</span>
          {{ t('sdk.import') }}
        </button>
      </div>
    </div>

    <div class="sync-notice">
      <span class="material-symbols-outlined">info</span>
      <div>
        <strong>{{ t('sync.notice_title') }}</strong>
        <p>{{ t('sync.notice_body') }}</p>
      </div>
    </div>

    <section class="sync-preview-panel">
      <div class="sync-preview-header">
        <h3>{{ t('sync.preview') }}</h3>
        <button class="btn text small" :disabled="loadingPreview" @click="loadPreview">
          <span class="material-symbols-outlined">refresh</span>
          {{ t('sync.refresh') }}
        </button>
      </div>
      <div v-if="loadingPreview" class="sync-preview-loading">
        <div class="spinner"></div>
        <span>{{ t('sync.loading') }}</span>
      </div>
      <pre v-else-if="previewText" class="sync-preview-code">{{ previewText }}</pre>
      <div v-else class="empty-state">{{ t('sync.empty') }}</div>
    </section>
  </div>
</template>
