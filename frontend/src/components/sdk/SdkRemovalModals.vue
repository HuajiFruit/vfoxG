<script lang="ts" setup>
import { t } from '../../i18n';
import ConfirmModal from '../common/ConfirmModal.vue';
import RemovePluginModal from './RemovePluginModal.vue';

type ConfirmAction = {
  type: 'removePlugin' | 'uninstallVersion' | 'removeCustomSdk' | null;
  name: string;
  version?: string;
  path?: string;
};

defineProps<{
  confirmAction: ConfirmAction;
  removePluginName: string | null;
  removePluginCustomSdks: Array<{ path: string; version: string }>;
}>();

const emit = defineEmits([
  'confirm-action',
  'cancel-action',
  'confirm-remove-plugin',
  'cancel-remove-plugin',
]);
</script>

<template>
  <Teleport to="body">
    <ConfirmModal
      v-if="confirmAction.type"
      :title="confirmAction.type === 'uninstallVersion' ? t('sdk.uninstall') : t('sdk.remove')"
      :message="confirmAction.type === 'uninstallVersion'
        ? t('sdk.confirm.uninstall_version_message', { name: confirmAction.name, version: confirmAction.version || '' })
        : t('sdk.confirm.remove_custom_message', { note: t('sdk.confirm.note'), question: t('sdk.confirm.remove_custom') })"
      :danger="true"
      :confirmText="confirmAction.type === 'uninstallVersion' ? t('sdk.uninstall') : t('sdk.remove_reference')"
      @confirm="emit('confirm-action')"
      @cancel="emit('cancel-action')"
    />
    <RemovePluginModal
      v-if="removePluginName"
      :pluginName="removePluginName"
      :customSdks="removePluginCustomSdks"
      @confirm="emit('confirm-remove-plugin', $event)"
      @cancel="emit('cancel-remove-plugin')"
    />
  </Teleport>
</template>
