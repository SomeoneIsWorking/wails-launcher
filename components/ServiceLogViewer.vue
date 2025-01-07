<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <div
      class="absolute z-10 top-4 right-6 flex items-stretch bg-gray-800/90 rounded-full shadow-lg backdrop-blur-sm text-sm text-gray-300"
    >
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
      :items="service.logs"
      :item-height="24"
      height="100%"
      :buffer="10"
      @scroll="handleScroll"
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
import { ref, nextTick, computed, onMounted } from "vue";
import { ChevronUp, ChevronDown } from "lucide-vue-next";

const props = defineProps<{
  serviceId: string;
}>();
const store = useServicesStore();
const service = computed(() => store.services[props.serviceId]);
const virtualScroller = ref();
const isScrolledToBottom = ref(true);
const currentOrPreviousErrorIndex = ref(-1);

const errors = computed(() =>
  service.value.logs
    .map((log, index) => ({ ...log, elementIndex: index }))
    .filter(({ level }) => level === "ERR")
);

const errorsAbove = ref<typeof errors.value>([]);
const errorsBelow = ref<typeof errors.value>([]);

const handleScroll = ({
  scrollTop,
  isAtBottom,
}: {
  scrollTop: number;
  isAtBottom: boolean;
}) => {
  isScrolledToBottom.value = isAtBottom;

  // Update errors above/below based on scroll position
  const viewportHeight = virtualScroller.value?.$el.clientHeight ?? 0;
  errorsAbove.value = errors.value.filter(
    (x) => x.elementIndex * 24 < scrollTop
  );
  errorsBelow.value = errors.value.filter(
    (x) => x.elementIndex * 24 > scrollTop + viewportHeight
  );

  currentOrPreviousErrorIndex.value = errors.value.findLastIndex(
    (x) => x.elementIndex * 24 <= scrollTop + viewportHeight
  );
};

onMounted(() => {
  const savedPosition = store.getScrollPosition(props.serviceId);
  if (savedPosition) {
    virtualScroller.value?.scrollTo(savedPosition);
  } else {
    scrollToBottom();
  }
});

onBeforeUnmount(() => {
  store.saveScrollPosition(
    props.serviceId,
    isScrolledToBottom.value ? undefined : virtualScroller.value?.$el.scrollTop
  );
});

const scrollToBottom = () => {
  virtualScroller.value?.scrollToBottom();
};

const navigateError = (error: (typeof errors.value)[number] | undefined) => {
  if (!error) return;
  virtualScroller.value?.scrollTo(error.elementIndex * 24);
};

const clearLogs = async () => {
  await store.clearLogs(props.serviceId);
};

watch(
  () => service.value.logs,
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
</style>
