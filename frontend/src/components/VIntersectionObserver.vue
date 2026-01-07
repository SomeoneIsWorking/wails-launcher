<template>
  <div ref="element">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  threshold?: number;
}>();

const emit = defineEmits<{
  intersection: [];
}>();

const element = ref<HTMLElement | null>(null);

onMounted(() => {
  const observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting) {
        emit("intersection");
      }
    },
    { threshold: props.threshold ?? 0.5 }
  );

  observer.observe(element.value!);

  onUnmounted(() => {
    observer.disconnect();
  });
});
</script>
