<template>
  <slot ref="element" />
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from "vue";

const props = defineProps<{
  threshold?: number;
  enabled?: boolean;
}>();

const emit = defineEmits<{
  intersection: [];
}>();

const element = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

const setupObserver = () => {
  if (!element.value || !props.enabled) {
    return;
  }

  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) {
        emit("intersection");
      }
    },
    { threshold: props.threshold ?? 0.5 }
  );

  observer.observe(element.value);
};

const cleanupObserver = () => {
  if (observer) {
    observer.disconnect();
    observer = null;
  }
};

onMounted(() => {
  setupObserver();
});

watch(
  () => props.enabled,
  (enabled) => {
    if (enabled) {
      setupObserver();
    } else {
      cleanupObserver();
    }
  }
);

onUnmounted(() => {
  cleanupObserver();
});
</script>
