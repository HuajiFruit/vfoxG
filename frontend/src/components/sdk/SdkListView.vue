<script lang="ts" setup>
import type { SdkInfo } from '../../services/appModels';
import { t } from '../../i18n';
import PluginIcon from '../plugin/PluginIcon.vue';
import SdkEmptyState from './SdkEmptyState.vue';

defineProps<{
  loading: boolean;
  sdks: SdkInfo[];
  vfoxSdks: SdkInfo[];
  systemSdks: SdkInfo[];
  getTotalVersionCount: (sdk: SdkInfo) => number;
  displayVersion: (version?: string) => string;
  truncateVersion: (version?: string) => string;
}>();

const emit = defineEmits(['open-detail', 'open-plugin-market', 'open-sync']);
</script>

<template>
  <div class="view-container">
    <div class="sdk-list-header">
      <h2>{{ t('sdk.installed.title') }}</h2>
    </div>

    <div v-if="loading" class="flex-center" style="height: 200px;">
      <div class="spinner"></div>
    </div>

    <SdkEmptyState
      v-else-if="sdks.length === 0"
      show-steps
      @open-plugin-market="emit('open-plugin-market')"
      @open-sync="emit('open-sync')"
    />

    <template v-else>
      <template v-if="vfoxSdks.length">
        <h3 class="section-heading">{{ t('sdk.vfox.title') }}</h3>
        <div class="sdk-grid">
          <div v-for="sdk in vfoxSdks" :key="sdk.name" class="sdk-card" @click="emit('open-detail', sdk)">
            <PluginIcon :name="sdk.name" class="sdk-icon-large" />
            <div class="sdk-card-content">
              <h3>{{ sdk.name }}</h3>
              <span class="version-count">
                {{ getTotalVersionCount(sdk) }}
                {{ getTotalVersionCount(sdk) !== 1 ? t('sdk.versions') : t('sdk.version') }}
              </span>
            </div>
          </div>
        </div>
      </template>

      <template v-if="systemSdks.length">
        <h3 class="section-heading" style="margin-top: 32px;">{{ t('sdk.nonevfox.title') }}</h3>
        <div class="sdk-grid">
          <div v-for="sdk in systemSdks" :key="sdk.name" class="sdk-card card-system" @click="emit('open-detail', sdk)">
            <PluginIcon :name="sdk.name" class="sdk-icon-large" />
            <div class="sdk-card-content">
              <h3>{{ sdk.name }}</h3>
              <span class="version-count" :title="displayVersion(sdk.versions?.[0]?.version)">
                {{ truncateVersion(sdk.versions?.[0]?.version) }}
              </span>
            </div>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>
