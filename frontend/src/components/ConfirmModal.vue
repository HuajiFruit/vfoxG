<script lang="ts" setup>
defineProps<{
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
}>();

const emit = defineEmits(['confirm', 'cancel']);
</script>

<template>
  <Transition name="modal" appear>
    <div class="modal-overlay" @click="emit('cancel')">
      <div class="modal-content" @click.stop>
        <h2 class="modal-title">{{ title }}</h2>
        <p class="modal-message">{{ message }}</p>
        <div class="modal-actions">
          <button class="btn tonal" @click="emit('cancel')">{{ cancelText || 'Cancel' }}</button>
          <button class="btn primary" :style="danger ? 'background: var(--md-error);' : ''" @click="emit('confirm')">{{ confirmText || 'Confirm' }}</button>
        </div>
      </div>
    </div>
  </Transition>
</template>
