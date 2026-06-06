<script lang="ts" setup>
import { ref } from 'vue';
import { t } from '../../i18n';
import type { TerminalLogEntry } from '../../composables/useTaskTerminal';

defineProps<{
  visible: boolean;
  logs: TerminalLogEntry[];
}>();

const emit = defineEmits(['clear', 'hide']);
const terminalBody = ref<HTMLElement | null>(null);

const scrollToBottom = () => {
  if (!terminalBody.value) return;
  terminalBody.value.scrollTop = terminalBody.value.scrollHeight;
};

defineExpose({
  scrollToBottom,
});
</script>

<template>
  <Transition name="terminal-dock-slide">
    <section v-if="visible" class="terminal-dock" :aria-label="t('terminal.aria')">
      <div class="terminal-header">
        <div class="terminal-title">
          <span class="material-symbols-outlined">terminal</span>
          {{ t('terminal.title') }}
          <span class="terminal-count">{{ logs.length }}</span>
        </div>
        <div class="terminal-actions">
          <button class="terminal-icon-btn" :title="t('terminal.clear')" @click="emit('clear')">
            <span class="material-symbols-outlined">delete_sweep</span>
          </button>
          <button class="terminal-icon-btn" :title="t('terminal.hide')" @click="emit('hide')">
            <span class="material-symbols-outlined">keyboard_arrow_down</span>
          </button>
        </div>
      </div>
      <div ref="terminalBody" class="terminal-body">
        <div v-if="logs.length === 0" class="terminal-empty">{{ t('terminal.empty') }}</div>
        <div v-for="entry in logs" :key="entry.id" class="terminal-line" :class="entry.level">
          <span class="terminal-time">{{ entry.time }}</span>
          <span class="terminal-prompt">$</span>
          <span class="terminal-text">{{ entry.text }}</span>
        </div>
      </div>
    </section>
  </Transition>
</template>
