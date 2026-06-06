<script lang="ts" setup>
import type { PluginInfo } from '../../services/appModels';
import { t } from '../../i18n';
import { getPluginDescription } from '../../pluginDescriptions';
import PluginIcon from './PluginIcon.vue';

defineProps<{
  plugin: PluginInfo;
  availableVersions: string[];
  installedVersions: Set<string>;
  loadingVersions: boolean;
  installingVersion: string | null;
  addingPlugin: string | null;
  removingPlugin: string | null;
}>();

const emit = defineEmits([
  'back',
  'open-url',
  'remove-plugin',
  'add-plugin',
  'install-version',
]);
</script>

<template>
  <div class="view-container detail-view">
    <div class="detail-header">
      <button class="btn tonal small back-btn" @click="emit('back')">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
        {{ t('sdk.back') }}
      </button>

      <div class="detail-title-area" style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
        <div style="display: flex; align-items: center; gap: 24px;">
          <PluginIcon :name="plugin.name" class="sdk-icon-huge" />
          <div class="detail-title-text">
            <h1>{{ plugin.name }}</h1>
            <p v-if="getPluginDescription(plugin.name)" style="color: var(--text-secondary); margin: 4px 0 8px 0; font-size: 14px; max-width: 500px; line-height: 1.5;">{{ getPluginDescription(plugin.name) }}</p>
            <button class="link" @click="emit('open-url', plugin.url)">{{ t('market.homepage') }} &nearr;</button>
          </div>
        </div>

        <div v-if="plugin.isAdded">
          <button
            class="btn tonal small"
            style="color: var(--md-error); background: rgba(239, 68, 68, 0.1); min-width: 120px; display: flex; justify-content: center; align-items: center;"
            :disabled="removingPlugin === plugin.name"
            @click="emit('remove-plugin', plugin.name)"
          >
            <div v-if="removingPlugin === plugin.name" class="spinner small-spinner" style="width: 16px; height: 16px; border-width: 2px; border-top-color: var(--md-error);"></div>
            <template v-else>
              <span class="material-symbols-outlined" style="font-size: 16px; margin-right: 4px;">delete</span>
              {{ t('sdk.remove_plugin') }}
            </template>
          </button>
        </div>
      </div>
    </div>

    <div class="detail-body">
      <div v-if="!plugin.isAdded" class="plugin-not-added-banner flex-center" style="flex-direction: column; padding: 60px 20px; text-align: center;">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" stroke-width="1.5" style="margin-bottom: 16px;"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
        <h2 style="margin-bottom: 8px;">{{ t('market.plugin_not_added') }}</h2>
        <p style="color: var(--text-secondary); margin-bottom: 24px; max-width: 400px;">
          {{ t('market.plugin_not_added.desc') }}
        </p>
        <button class="btn primary large-btn" :disabled="addingPlugin === plugin.name" style="min-width: 140px; display: flex; justify-content: center; align-items: center;" @click="emit('add-plugin', plugin.name)">
          <div v-if="addingPlugin === plugin.name" class="spinner small-spinner" style="width: 18px; height: 18px; border-width: 2px;"></div>
          <template v-else>+ {{ t('market.add_plugin') }}</template>
        </button>
      </div>

      <div v-else>
        <h2>{{ t('sdk.available_versions') }}</h2>
        <p style="color: var(--text-secondary); margin-bottom: 20px;">{{ t('market.select_version_hint') }}</p>

        <div v-if="loadingVersions" class="flex-center" style="height: 200px;">
          <div class="spinner"></div>
        </div>

        <div v-else-if="availableVersions.length === 0" class="empty-state">
          {{ t('market.no_versions') }}
        </div>

        <div v-else class="search-results-grid">
          <div v-for="version in availableVersions" :key="version" class="search-result-card" style="cursor: default;">
            <span>{{ version }}</span>
            <div class="plugin-actions">
              <span v-if="installedVersions.has(version)" class="btn tonal small" style="pointer-events: none; width: 96px; padding: 0; display: flex; justify-content: center; align-items: center; color: #34d399; font-weight: 600; text-transform: uppercase;">{{ t('market.installed') }}</span>
              <button
                v-else
                class="btn tonal small"
                :disabled="installingVersion === version"
                style="width: 96px; padding: 0;"
                @click="emit('install-version', plugin.name, version)"
              >
                {{ installingVersion === version ? '...' : t('market.install') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
