<template>
  <div
    ref="scrollerRef"
    class="flex-1 overflow-y-auto font-mono text-sm bg-gray-900 text-gray-100 relative"
    @scroll="handleScroll"
  >
    <div :style="{ height: totalHeight + 'px' }" class="p-4 relative">
      <div
        v-for="item in visibleItems"
        :key="item.index"
        :data-index="item.index"
        class="w-full absolute"
        :style="{ transform: `translateY(${item.top}px)` }"
      >
        <slot :item="item.data" :index="item.index" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ClientLogEntry } from "@/types/client";
import { ref, computed, onMounted, watch, nextTick } from "vue";

const props = withDefaults(
  defineProps<{
    items: ClientLogEntry[];
    buffer?: number;
  }>(),
  {
    buffer: 5,
  }
);

const emit = defineEmits<{
  scroll: [{ scrollTop: number; isAtBottom: boolean }];
  ready: [];
}>();

const scrollerRef = ref<HTMLElement | null>(null);
const scrollTop = ref(0);

const visibleItems = ref<
  Array<{
    index: number;
    data: ClientLogEntry;
    top: number;
  }>
>([]);

const totalHeight = computed(() => {
  if (!scrollerRef.value) {
    return 0;
  }

  return props.items.reduce((acc, item) => {
    return acc + item.height;
  }, 0);
});

const updatePool = () => {
  if (!scrollerRef.value || props.items.length === 0) {
    visibleItems.value = [];
    return;
  }

  const currentScroll = Math.max(0, scrollTop.value);
  let i = 0;
  let accHeight = 0;

  const items = [];

  // Before visible start
  while (i < props.items.length) {
    const height = props.items[i].height;
    items.push({
      index: i,
      data: props.items[i],
      top: accHeight,
    });
    i++;
    if (items.length > props.buffer) {
      items.shift();
    }
    accHeight += height;
    if (accHeight > currentScroll) {
      break;
    }
  }

  // Visible items
  while (i < props.items.length) {
    const height = props.items[i].height;
    items.push({
      index: i,
      data: props.items[i],
      top: accHeight,
    });
    accHeight += height;
    i++;

    if (accHeight > currentScroll + scrollerRef.value.clientHeight) {
      break;
    }
  }

  // After visible end
  const visibleEnd = i + props.buffer;
  while (i < props.items.length && i < visibleEnd) {
    const height = props.items[i].height;
    items.push({
      index: i,
      data: props.items[i],
      top: accHeight,
    });
    accHeight += height;
    i++;
  }

  visibleItems.value = items;
};

const handleScroll = () => {
  if (!scrollerRef.value) {
    return;
  }
  scrollTop.value = scrollerRef.value.scrollTop;
  updatePool();

  const { scrollHeight, clientHeight } = scrollerRef.value;
  emit("scroll", {
    scrollTop: scrollTop.value,
    isAtBottom: scrollHeight - scrollTop.value - clientHeight < 10,
  });
};

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

const getItemPosition = (index: number): number => {
  let position = 0;
  for (let i = 0; i < index; i++) {
    position += props.items[i].height;
  }
  return position;
};

const scrollToIndex = (index: number, offset: number) => {
  if (!scrollerRef.value) {
    return;
  }

  const position = getItemPosition(index);
  scrollerRef.value.scrollTop = position + offset;
};

const getVisibleRange = () => {
  if (!scrollerRef.value) {
    return { start: 0, end: 0 };
  }

  const containerHeight = scrollerRef.value.clientHeight;
  const currentScroll = scrollTop.value;

  let visibleStart = 0;
  let accHeight = 0;

  for (let i = 0; i < props.items.length; i++) {
    const height = props.items[i].height;
    if (accHeight + height > currentScroll) {
      visibleStart = i;
      break;
    }
    accHeight += height;
  }

  // Find last visible item
  let visibleEnd = visibleStart;
  let heightSum = 0;

  while (visibleEnd < props.items.length && heightSum < containerHeight) {
    heightSum += props.items[visibleEnd].height;
    visibleEnd++;
  }

  return {
    start: visibleStart,
    end: visibleEnd - 1,
  };
};

defineExpose({
  scrollTo,
  scrollToBottom,
  scrollToIndex,
  getVisibleRange,
  getItemPosition,
});

onMounted(() => {
  updatePool();
  // Emit ready event after initial pool update
  nextTick(() => {
    emit("ready");
  });
});

// Watch items array for any changes (additions, removals, or replacements)
watch(
  () => props.items.length,
  () => {
    // Ensure scroll position is valid after items change
    if (scrollerRef.value) {
      const maxScroll = Math.max(
        0,
        scrollerRef.value.scrollHeight - scrollerRef.value.clientHeight
      );
      if (scrollTop.value > maxScroll) {
        scrollTop.value = maxScroll;
        scrollerRef.value.scrollTop = maxScroll;
      }
    }
    updatePool();
  }
);
</script>
