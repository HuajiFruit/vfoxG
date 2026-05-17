<script lang="ts" setup>
import { ref, computed, watch } from 'vue';

const props = defineProps<{
  name: string;
}>();

const hasError = ref(false);

watch(() => props.name, () => {
  hasError.value = false;
});

const iconUrl = computed(() => {
  return `/icons/${props.name.toLowerCase()}.svg`;
});

const fallbackColor = computed(() => {
  let hash = 0;
  for (let i = 0; i < props.name.length; i++) {
    hash = props.name.charCodeAt(i) + ((hash << 5) - hash);
  }
  const h = Math.abs(hash) % 360;
  return `hsl(${h}, 55%, 60%)`;
});

const handleImageError = () => {
  hasError.value = true;
};
</script>

<template>
  <img 
    v-if="!hasError" 
    :src="iconUrl" 
    class="plugin-icon svg-icon" 
    @error="handleImageError" 
    alt="plugin icon" 
  />
  <span v-else class="plugin-icon fallback-letter" :style="{ color: fallbackColor, background: fallbackColor + '1A' }">
    {{ name.charAt(0).toUpperCase() }}
  </span>
</template>

<style scoped>
.plugin-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--md-on-surface);
}

.svg-icon {
  object-fit: contain;
}

.fallback-letter {
  font-family: 'Roboto', sans-serif;
  font-size: 16px;
  font-weight: 700;
}
</style>
