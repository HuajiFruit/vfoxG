<script lang="ts" setup>
import type { PendingMigrationAction } from '../../composables/downloadPathMigration';
import { t } from '../../i18n';

defineProps<{
  pendingMigration: PendingMigrationAction | null;
  isDownloadPathBusy: boolean;
}>();

const emit = defineEmits(['confirm', 'cancel']);
</script>

<template>
  <Teleport to="body">
    <Transition name="modal" appear>
      <div
        v-if="pendingMigration"
        class="modal-overlay migration-confirm-overlay"
        role="dialog"
        aria-modal="true"
        @click="emit('cancel')"
      >
        <div class="modal-content migration-confirm-modal" @click.stop>
          <div class="migration-confirm-heading">
            <div class="migration-confirm-icon">
              <span class="material-symbols-outlined">sync_alt</span>
            </div>
            <div>
              <h2 class="modal-title">{{ t('settings.download.path.migrate_title') }}</h2>
              <p class="modal-message">{{ t('settings.download.path.migrate_confirm') }}</p>
            </div>
          </div>
          <div class="migration-confirm-path">
            <span>{{ t('settings.download.path.migrate_target') }}</span>
            <code>{{ pendingMigration.targetPath }}</code>
          </div>
          <div class="modal-actions">
            <button class="btn tonal" :disabled="isDownloadPathBusy" @click="emit('cancel')">
              {{ t('common.cancel') }}
            </button>
            <button class="btn primary" :disabled="isDownloadPathBusy" @click="emit('confirm')">
              <div v-if="isDownloadPathBusy" class="spinner small-spinner"></div>
              <template v-else>{{ t('common.confirm') }}</template>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
