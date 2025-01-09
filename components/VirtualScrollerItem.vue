<template>
  <div ref="elRef" :data-index="index">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';

const props = defineProps<{
  index: number;
}>();

const emit = defineEmits<{
  resize: [{ height: number; index: number }];
}>();

const elRef = ref<HTMLElement | null>(null);
let observer: ResizeObserver | null = null;

onMounted(() => {
  if (!elRef.value) return;
  
  observer = new ResizeObserver((entries) => {
    const height = entries[0].borderBoxSize[0]?.blockSize || 
                  (entries[0].target as HTMLElement).offsetHeight;
    emit('resize', { height, index: props.index });
  });
  
  observer.observe(elRef.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
});
</script>
