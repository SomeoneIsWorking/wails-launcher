<template>
  <div
    ref="scrollerRef"
    class="flex-1 overflow-y-auto font-mono text-sm bg-gray-900 text-gray-100 relative"
    @scroll="handleScroll"
    :style="{ height }"
  >
    <div :style="{ height: totalHeight + 'px' }" class="absolute p-4 inset-x-0">
      <div
        :style="{
          transform: `translateY(${startOffset}px)`,
        }"
      >
        <slot
          v-for="item in visibleItems"
          :key="item.index"
          :item="item.data"
          :index="item.index"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

const props = defineProps<{
  items: any[];
  itemHeight: number;
  height: string;
  buffer?: number;
}>();

const emit = defineEmits<{
  scroll: [{ scrollTop: number; isAtBottom: boolean }];
}>();

const scrollerRef = ref<HTMLElement | null>(null);
const scrollTop = ref(0);

const handleScroll = () => {
  if (!scrollerRef.value) return;
  scrollTop.value = scrollerRef.value.scrollTop;
  const { scrollHeight, clientHeight } = scrollerRef.value;
  emit("scroll", {
    scrollTop: scrollTop.value,
    isAtBottom: scrollHeight - scrollTop.value - clientHeight < 10,
  });
};

const visibleRange = computed(() => {
  if (!scrollerRef.value) return { start: 0, end: 10 };

  const buffer = props.buffer || 5;
  const containerHeight = scrollerRef.value.clientHeight;
  const startIndex = Math.max(
    0,
    Math.floor(scrollTop.value / props.itemHeight) - buffer
  );
  const endIndex = Math.min(
    props.items.length,
    Math.ceil((scrollTop.value + containerHeight) / props.itemHeight) + buffer
  );

  return { start: startIndex, end: endIndex };
});

const visibleItems = computed(() => {
  return props.items
    .slice(visibleRange.value.start, visibleRange.value.end)
    .map((item, index) => ({
      data: item,
      index: index + visibleRange.value.start,
    }));
});

const startOffset = computed(() => visibleRange.value.start * props.itemHeight);

const totalHeight = computed(() => props.items.length * props.itemHeight);

const scrollTo = (position: number) => {
  if (scrollerRef.value) {
    scrollerRef.value.scrollTop = position;
  }
};

const scrollToBottom = () => {
  if (scrollerRef.value) {
    scrollerRef.value.scrollTop = scrollerRef.value.scrollHeight;
  }
};

defineExpose({
  scrollTo,
  scrollToBottom,
});
</script>
