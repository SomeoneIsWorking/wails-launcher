<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <div
      class="absolute top-4 right-4 flex items-stretch bg-gray-800/90 rounded-full shadow-lg backdrop-blur-sm text-sm text-gray-300"
    >
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
    <div
      ref="logContainer"
      @scroll="handleScroll"
      class="flex-1 overflow-y-auto font-mono text-sm p-4 bg-gray-900 text-gray-100"
    >
      <LogEntry
        v-for="(log, index) in service.logs"
        :key="index"
        :log="log"
        :service-name="service.name"
      />
    </div>

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
import { ref, watch, nextTick, computed, onMounted } from "vue";
import type { ClientServiceInfo } from "~/types/client";
import { ChevronUp, ChevronDown } from "lucide-vue-next";

const props = defineProps<{
  service: ClientServiceInfo;
}>();

const logContainer = ref<HTMLElement | null>(null);
const isScrolledToBottom = ref(true);
const currentOrPreviousErrorIndex = ref(-1);

const errors = computed(() =>
  props.service.logs
    .map((log, index) => ({ ...log, elementIndex: index }))
    .filter(({ level }) => level === "ERR")
    .map((log) => ({
      ...log,
      element: () =>
        logContainer.value?.children[log.elementIndex] as
          | HTMLElement
          | undefined,
    }))
);

type ErrorLog = (typeof errors)["value"][number];

const errorsAbove = ref<ErrorLog[]>([]);
const errorsBelow = ref<ErrorLog[]>([]);

const handleScroll = () => {
  if (!logContainer.value) return;

  const { scrollTop, scrollHeight, clientHeight } = logContainer.value;
  const scrollRect = {
    top: scrollTop,
    bottom: scrollTop + clientHeight,
  };
  isScrolledToBottom.value = scrollHeight - scrollTop - clientHeight < 10;
  errorsAbove.value = errors.value.filter((x) => {
    const element = x.element();
    if (!element) return false;
    return element.offsetTop < scrollTop;
  });
  errorsBelow.value = errors.value.filter((x) => {
    const element = x.element();
    if (!element) return false;
    return element.offsetTop > scrollRect.top;
  });
  currentOrPreviousErrorIndex.value = errors.value.findLastIndex((x) => {
    const element = x.element();
    if (!element) return false;
    const rect = {
      top: element.offsetTop,
      bottom: element.offsetTop + element.offsetHeight,
    };
    return rect.top <= scrollRect.bottom;
  });
};

onMounted(() => {
  handleScroll();
});

const scrollToBottom = () => {
  logContainer.value?.scrollTo({
    top: logContainer.value.scrollHeight,
    behavior: "smooth",
  });
};

const navigateError = (item: ErrorLog | undefined) => {
  const targetEl = item?.element();
  if (!targetEl) return;
  targetEl.parentElement?.scroll({
    top: targetEl.offsetTop,
    behavior: "smooth",
  });
};

watch(
  () => props.service.logs,
  () => {
    handleScroll();
    if (isScrolledToBottom.value) {
      nextTick(scrollToBottom);
    }
  },
  { deep: true }
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
