<template>
  <div
    ref="scrollerRef"
    class="flex-1 overflow-y-auto font-mono text-sm bg-gray-900 text-gray-100 relative break-words"
    @scroll="handleScroll"
    :style="{ height }"
  >
    <div :style="{ height: totalHeight + 'px' }" class="absolute p-4 inset-x-0">
      <template v-for="pool in sizingPools" :key="pool.id">
        <div
          :style="{
            position: 'absolute',
            top: 0,
            transform: `translateY(${pool.top}px)`,
            width: '100%',
          }"
        >
          <VirtualScrollerItem
            v-for="item in pool.items"
            :key="item.id"
            :index="item.index"
            @resize="handleResize"
          >
            <slot
              :item="item.data"
              :index="item.index"
            />
          </VirtualScrollerItem>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from "vue";

const props = defineProps<{
  items: any[];
  height: string;
  poolSize?: number;
  buffer?: number;
}>();

const emit = defineEmits<{
  scroll: [{ scrollTop: number; isAtBottom: boolean }];
  ready: [];
}>();

const scrollerRef = ref<HTMLElement | null>(null);
const scrollTop = ref(0);
const heightCache = new Map<number, number>();
const defaultHeight = 50;
const poolSize = props.poolSize || 2;

const sizingPools = ref<Array<{
  id: string;
  items: Array<{ id: number; index: number; data: any }>;
  top: number;
}>>([]);

const totalHeight = computed(() => {
  if (!scrollerRef.value) return 0;
  
  const baseHeight = props.items.reduce((acc, _, index) => {
    return acc + (heightCache.get(index) || defaultHeight);
  }, 0);
  
  const containerHeight = scrollerRef.value.clientHeight;
  const currentScroll = scrollTop.value;
  
  // If we're scrolled to the bottom, adjust the total height to prevent empty space
  if (currentScroll + containerHeight > baseHeight) {
    return currentScroll + containerHeight;
  }
  
  return baseHeight;
});

const updatePool = () => {
  if (!scrollerRef.value) return;

  const containerHeight = scrollerRef.value.clientHeight;
  const currentScroll = scrollTop.value;
  let accHeight = 0;
  let visibleStart = 0;

  // Find start index
  for (let i = 0; i < props.items.length; i++) {
    const height = heightCache.get(i) || defaultHeight;
    if (accHeight + height > currentScroll) {
      visibleStart = Math.max(0, i - 5);
      break;
    }
    accHeight += height;
  }

  // Create pools
  const newPools = [];
  let poolStartIndex = visibleStart;
  let poolTop = getItemsHeight(0, visibleStart);

  for (let i = 0; i < poolSize; i++) {
    const poolItems = props.items
      .slice(poolStartIndex, poolStartIndex + Math.ceil(containerHeight / defaultHeight))
      .map((item, idx) => ({
        id: poolStartIndex + idx,
        index: poolStartIndex + idx,
        data: item,
      }));

    if (poolItems.length === 0) break;

    newPools.push({
      id: `pool-${i}`,
      items: poolItems,
      top: poolTop,
    });

    poolTop += getItemsHeight(poolStartIndex, poolStartIndex + poolItems.length);
    poolStartIndex += poolItems.length;
  }

  sizingPools.value = newPools;
};

const handleScroll = () => {
  if (!scrollerRef.value) return;
  scrollTop.value = scrollerRef.value.scrollTop;
  updatePool();

  const { scrollHeight, clientHeight } = scrollerRef.value;
  emit("scroll", {
    scrollTop: scrollTop.value,
    isAtBottom: scrollHeight - scrollTop.value - clientHeight < 10,
  });
};

const handleResize = ({ height, index }: { height: number; index: number }) => {
  if (heightCache.get(index) !== height) {
    heightCache.set(index, height);
    updatePool();
  }
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
    position += heightCache.get(i) || defaultHeight;
  }
  return position;
};

const scrollToIndex = (index: number) => {
  if (!scrollerRef.value) return;

  const position = getItemPosition(index);
  scrollerRef.value.scrollTop = position;

  // Check if we actually reached the correct position after a short delay
  // to allow for height measurements to be updated
  setTimeout(() => {
    const currentPosition = getItemPosition(index);
    if (Math.abs(scrollerRef.value!.scrollTop - currentPosition) > 1) {
      // If we didn't reach the correct position, try again
      scrollerRef.value!.scrollTop = currentPosition;
    }
  }, 100);
};

const getVisibleRange = () => {
  if (!scrollerRef.value) return { start: 0, end: 0 };
  
  const containerHeight = scrollerRef.value.clientHeight;
  const currentScroll = scrollTop.value;
  
  // Find first visible item without buffer
  let visibleStart = 0;
  let accHeight = 0;
  
  for (let i = 0; i < props.items.length; i++) {
    const height = heightCache.get(i) || defaultHeight;
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
    heightSum += heightCache.get(visibleEnd) || defaultHeight;
    visibleEnd++;
  }

  return {
    start: visibleStart,
    end: visibleEnd - 1
  };
};

defineExpose({
  scrollTo,
  scrollToBottom,
  scrollToIndex,
  getVisibleRange,
  getItemPosition
});

const getItemsHeight = (start: number, end: number): number => {
  let height = 0;
  for (let i = start; i < end; i++) {
    height += heightCache.get(i) || defaultHeight;
  }
  return height;
};

onMounted(() => {
  updatePool();
  // Emit ready event after initial pool update
  nextTick(() => {
    emit('ready');
  });
});

watch(() => props.items.length, updatePool);
watch(() => scrollTop.value, updatePool);
</script>
