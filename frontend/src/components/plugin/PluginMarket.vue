<script lang="ts" setup>
import { inject } from 'vue';
import RemovePluginModal from '../sdk/RemovePluginModal.vue';
import PluginDetailView from './PluginDetailView.vue';
import PluginListView from './PluginListView.vue';
import { t } from '../../i18n';
import { useNotifyBridge } from '../../composables/useNotifyBridge';
import { usePluginMarketActions } from '../../composables/usePluginMarketActions';
import { usePluginMarketLifecycle } from '../../composables/usePluginMarketLifecycle';
import { usePluginMarketList } from '../../composables/usePluginMarketList';
import { usePluginVersionDetail } from '../../composables/usePluginVersionDetail';
import { useSdkTaskRunner, type RunTerminalTask } from '../../composables/useSdkTaskRunner';
import {
  openExternalUrl,
} from '../../services/pluginMarketService';

const emit = defineEmits(['start-task', 'notify']);
const runTerminalTask = inject<RunTerminalTask>('runTerminalTask');

const {
  notifyError,
  notifyInfo,
  getErrorMessage,
} = useNotifyBridge({
  emitNotify: payload => emit('notify', payload),
});

const {
  runTask,
  notifyTaskError,
} = useSdkTaskRunner({
  runTerminalTask,
  emitStartTask: (title: string) => emit('start-task', title),
  notifyInfo,
  notifyError,
  formatError: getErrorMessage,
});

const {
  activeView,
  transitionName,
  selectedPlugin,
  availableVersions,
  installedVersions,
  loadingVersions,
  fetchPluginVersions,
  openDetail,
  closeDetail,
  resetDetailImmediately,
  disposePluginVersionDetail,
} = usePluginVersionDetail({
  notifyError,
  formatError: getErrorMessage,
});

const {
  plugins,
  loading,
  officialPlugins,
  communityPlugins,
  fetchPlugins,
  invalidatePluginRequests,
} = usePluginMarketList({
  selectedPlugin,
  notifyError,
  formatError: getErrorMessage,
});

const openUrl = async (url: string) => {
  try {
    await openExternalUrl(url);
  } catch (err) {
    notifyError(getErrorMessage(err, t('market.open_error')));
  }
};

const {
  addingPlugin,
  removingPlugin,
  installingVersion,
  confirmRemove,
  confirmRemoveCustomSdks,
  addPlugin,
  requestRemovePlugin,
  executeRemovePlugin,
  installVersion,
} = usePluginMarketActions({
  selectedPlugin,
  installedVersions,
  runTask,
  notifyError,
  notifyTaskError,
  formatError: getErrorMessage,
  fetchPlugins,
  fetchPluginVersions,
});

usePluginMarketLifecycle({
  fetchPlugins,
  invalidatePluginRequests,
  resetDetailImmediately,
  disposePluginVersionDetail,
});
</script>

<template>
  <div class="plugin-market">

    <Transition :name="transitionName" mode="out-in">
      <PluginListView
        v-if="activeView === 'list'"
        key="list"
        :loading="loading"
        :official-plugins="officialPlugins"
        :community-plugins="communityPlugins"
        :adding-plugin="addingPlugin"
        @open-detail="openDetail"
        @open-url="openUrl"
        @add-plugin="addPlugin"
      />

      <PluginDetailView
        v-else-if="activeView === 'detail' && selectedPlugin"
        key="detail"
        :plugin="selectedPlugin"
        :available-versions="availableVersions"
        :installed-versions="installedVersions"
        :loading-versions="loadingVersions"
        :installing-version="installingVersion"
        :adding-plugin="addingPlugin"
        :removing-plugin="removingPlugin"
        @back="closeDetail"
        @open-url="openUrl"
        @remove-plugin="requestRemovePlugin"
        @add-plugin="addPlugin"
        @install-version="installVersion"
      />
    </Transition>

    <Teleport to="body">
      <RemovePluginModal
        v-if="confirmRemove"
        :pluginName="confirmRemove"
        :customSdks="confirmRemoveCustomSdks"
        @confirm="executeRemovePlugin"
        @cancel="confirmRemove = null"
      />
    </Teleport>
  </div>
</template>
