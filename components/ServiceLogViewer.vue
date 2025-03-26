<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <div
      class="absolute z-10 top-4 right-6 flex items-stretch bg-gray-800/90 rounded-full shadow-lg backdrop-blur-sm text-sm text-gray-300"
    >
      <div class="px-5 flex items-center">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search logs..."
          class="bg-transparent border-none outline-none text-white placeholder-gray-400 w-48"
        />
      </div>
      <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
      <button @click="clearLogs" class="px-5 py-2 hover:text-white">
        Clear Logs
      </button>
      <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
      <div class="px-5 flex items-center">
        Error {{ currentOrPreviousErrorIndex + 1 }} of {{ errors.length }}
      </div>
      <!-- vertical separator -->
      <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
      <div class="flex items-center gap-2 px-5 py-2">
        <button
          @click="navigateError(errorsAbove.at(-1))"
          :disabled="!errorsAbove.length"
          class="nav-buttons"
        >
          {{ errorsAbove.length }}
          <ChevronUp class="w-5 h-5" />
        </button>
        <button
          @click="navigateError(errorsBelow[0])"
          :disabled="!errorsBelow.length"
          class="nav-buttons"
        >
          {{ errorsBelow.length }}
          <ChevronDown class="w-5 h-5" />
        </button>
      </div>
    </div>
    <VirtualScroller
      ref="virtualScroller"
      :items="filteredLogs"
      height="100%"
      :buffer="10"
      @scroll="handleScroll"
      @ready="handleVirtualScrollerReady"
    >
      <template #default="{ item: log, index }">
        <LogEntry :log="log" :service-name="service.name" />
      </template>
    </VirtualScroller>
    <button
      v-if="!isScrolledToBottom"
      @click="scrollToBottom"
      class="fixed bottom-4 right-4 bg-blue-500 text-white p-2 rounded-full"
    >
      <ChevronDown class="w-5 h-5" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, computed, onMounted, watch, onBeforeUnmount } from "vue";
import { ChevronUp, ChevronDown } from "lucide-vue-next";
import type { ComponentInstance } from "vue";
import VirtualScroller from "./VirtualScroller.vue";
import type { ScrollPosition } from "~/types/client";

const props = defineProps<{
  serviceId: string;
}>();
const store = useServicesStore();
const service = computed(() => store.services[props.serviceId]);
const virtualScroller = ref<ComponentInstance<typeof VirtualScroller>>();
const isScrolledToBottom = ref(true);
const currentOrPreviousErrorIndex = ref(-1);

const errors = computed(() =>
  service.value.logs
    .map((log, index) => ({ ...log, elementIndex: index }))
    .filter(({ level }) => level === "ERR")
);

const errorsAbove = ref<typeof errors.value>([]);
const errorsBelow = ref<typeof errors.value>([]);

const searchQuery = ref("");

const updateErrorNavigation = () => {
  const range = virtualScroller.value?.getVisibleRange();
  if (!range) return;

  errorsAbove.value = errors.value.filter(
    error => error.elementIndex < range.start
  );
  
  errorsBelow.value = errors.value.filter(
    error => error.elementIndex > range.end
  );

  currentOrPreviousErrorIndex.value = errors.value.findIndex(
    error => error.elementIndex <= range.end && error.elementIndex >= range.start
  );
};

const filteredLogs = computed(() => {
  if (!searchQuery.value) return service.value.logs;
  
  const query = searchQuery.value.toLowerCase();
  return service.value.logs.filter(log => 
    log.message.toLowerCase().includes(query) ||
    log.level.toLowerCase().includes(query) ||
    log.timestamp.toLowerCase().includes(query)
  );
});

const handleScroll = ({
  scrollTop,
  isAtBottom,
}: {
  scrollTop: number;
  isAtBottom: boolean;
}) => {
  isScrolledToBottom.value = isAtBottom;
  updateErrorNavigation();
};

const savedPosition = ref<ScrollPosition | undefined>(undefined);

onMounted(() => {
  savedPosition.value = store.getScrollPosition(props.serviceId);
  if (!savedPosition.value) {
    scrollToBottom();
  }
});

const handleVirtualScrollerReady = () => {
  if (savedPosition.value !== undefined) {
    const { topIndex, offset } = savedPosition.value;
    virtualScroller.value?.scrollToIndex(topIndex);
    // After scrolling to the index, adjust by the offset
    nextTick(() => {
      if (virtualScroller.value?.$el) {
        virtualScroller.value.$el.scrollTop += offset;
      }
      updateErrorNavigation();
    });
    savedPosition.value = undefined;
  } else {
    updateErrorNavigation();
  }
};

onBeforeUnmount(() => {
  if (!virtualScroller.value?.$el) return;
  
  const range = virtualScroller.value.getVisibleRange();
  if (!range) return;
  
  const scrollTop = virtualScroller.value.$el.scrollTop;
  const firstItemTop = virtualScroller.value.getItemPosition(range.start);
  const offset = scrollTop - firstItemTop;
  
  store.saveScrollPosition(
    props.serviceId,
    isScrolledToBottom.value ? undefined : {
      topIndex: range.start,
      offset
    }
  );
});

const scrollToBottom = () => {
  virtualScroller.value?.scrollToBottom();
};

const navigateError = (error: (typeof errors.value)[number] | undefined) => {
  if (!error) return;
  virtualScroller.value?.scrollToIndex(error.elementIndex);
};

const clearLogs = async () => {
  await store.clearLogs(props.serviceId);
};

watch(
  () => filteredLogs.value,
  () => {
    if (isScrolledToBottom.value) {
      nextTick(scrollToBottom);
    }
  }
);
</script>

<style scoped lang="postcss">
:deep(a) {
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: underline;
}

button:not(:disabled):hover {
  background-color: #4b5563;
}

button.nav-buttons {
  @apply rounded-full bg-gray-700 text-white disabled:opacity-50 flex items-center justify-center h-6 w-12;
}

input::placeholder {
  color: #9ca3af;
}

input:focus::placeholder {
  color: #6b7280;
}
</style>
