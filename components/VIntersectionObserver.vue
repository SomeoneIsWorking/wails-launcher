<template>
  <div ref="element">
    <slot />
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  threshold?: number
  once?: boolean
}>()

const emit = defineEmits<{
  intersection: []
}>()

const element = ref<HTMLElement | null>(null)

onMounted(() => {
  const observer = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting) {
      emit('intersection')
      if (props.once && element.value) {
        observer.unobserve(element.value)
      }
    }
  }, { threshold: props.threshold ?? 0.5 })
  
  if (element.value) {
    observer.observe(element.value)
  }

  onUnmounted(() => {
    observer.disconnect()
  })
})
</script>
