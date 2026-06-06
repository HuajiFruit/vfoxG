<script lang="ts" setup>
import type { PluginInfo } from '../../services/appModels';
import { t } from '../../i18n';
import PluginIcon from './PluginIcon.vue';

defineProps<{
  loading: boolean;
  officialPlugins: PluginInfo[];
  communityPlugins: PluginInfo[];
  addingPlugin: string | null;
}>();

const emit = defineEmits(['open-detail', 'open-url', 'add-plugin']);
</script>

<template>
  <div class="view-container">
    <h2>{{ t('market.title') }}</h2>

    <div v-if="loading" class="flex-center" style="height: 200px; flex-direction: column; gap: 16px;">
      <div class="spinner"></div>
      <span style="color: var(--text-secondary); font-size: 14px;">{{ t('market.loading') }}</span>
    </div>

    <div v-else>
      <h3 v-if="officialPlugins.length > 0" class="section-heading">{{ t('market.official') }}</h3>
      <div class="sdk-grid">
        <div v-for="plugin in officialPlugins" :key="plugin.name" class="sdk-card" @click="emit('open-detail', plugin)">
          <PluginIcon :name="plugin.name" class="sdk-icon-large" />
          <div class="sdk-card-content" style="flex: 1;">
            <h3>{{ plugin.name }}</h3>
            <button class="link" style="font-size: 12px; margin-top: 4px; padding: 0;" @click.stop="emit('open-url', plugin.url)">
              {{ t('market.homepage') }} &nearr;
            </button>
          </div>
          <div class="plugin-actions">
            <span v-if="plugin.isAdded" class="btn tonal small" style="pointer-events: none; width: 80px; padding: 0; display: flex; justify-content: center; align-items: center; text-transform: uppercase; font-weight: 600;">{{ t('market.installed') }}</span>
            <button
              v-else
              class="btn primary small"
              :disabled="addingPlugin === plugin.name"
              style="width: 80px; padding: 0; display: flex; justify-content: center; align-items: center;"
              @click.stop="emit('add-plugin', plugin.name)"
            >
              <div v-if="addingPlugin === plugin.name" class="spinner small-spinner" style="width: 14px; height: 14px; border-width: 2px;"></div>
              <template v-else>{{ t('sdk.add') }}</template>
            </button>
          </div>
        </div>
      </div>

      <h3 v-if="communityPlugins.length > 0" class="section-heading" style="margin-top: 32px; color: var(--text-secondary);">{{ t('market.community') }}</h3>
      <div v-if="communityPlugins.length > 0" class="sdk-grid">
        <div v-for="plugin in communityPlugins" :key="plugin.name" class="sdk-card" @click="emit('open-detail', plugin)">
          <PluginIcon :name="plugin.name" class="sdk-icon-large" />
          <div class="sdk-card-content" style="flex: 1;">
            <h3>{{ plugin.name }}</h3>
            <button class="link" style="font-size: 12px; margin-top: 4px; padding: 0;" @click.stop="emit('open-url', plugin.url)">
              {{ t('market.homepage') }} &nearr;
            </button>
          </div>
          <div class="plugin-actions">
            <span v-if="plugin.isAdded" class="btn tonal small" style="pointer-events: none; width: 80px; padding: 0; display: flex; justify-content: center; align-items: center; text-transform: uppercase; font-weight: 600;">{{ t('market.installed') }}</span>
            <button
              v-else
              class="btn primary small"
              :disabled="addingPlugin === plugin.name"
              style="width: 80px; padding: 0; display: flex; justify-content: center; align-items: center;"
              @click.stop="emit('add-plugin', plugin.name)"
            >
              <div v-if="addingPlugin === plugin.name" class="spinner small-spinner" style="width: 14px; height: 14px; border-width: 2px;"></div>
              <template v-else>{{ t('sdk.add') }}</template>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
