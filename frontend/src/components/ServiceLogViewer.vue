<template>
  <div class="flex-1 flex flex-col overflow-y-hidden relative">
    <LogViewerControls
      v-model:search-query="searchQuery"
      :errors="errors"
      :errors-above="errorsAbove"
      :errors-below="errorsBelow"
      :current-or-previous-error-index="currentOrPreviousErrorIndex"
      @clear-logs="clearLogs"
      @navigate-error="navigateError"
    />
    <VirtualScroller
      ref="virtualScroller"
      :items="filteredLogs"
      @scroll="handleScroll"
      @ready="handleVirtualScrollerReady"
      class="text-[0.8rem] whitespace-pre leading-5"
    >
      <template #default="{ item: log, index: _index }">
        <LogEntry
          :log="log"
          :service-name="service.name"
          :service-path="service.path"
          :common-base-path="commonBasePath"
        />
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
import {
  ref,
  nextTick,
  computed,
  onMounted,
  watch,
  onBeforeUnmount,
} from "vue";
import { ChevronDown } from "lucide-vue-next";
import { useServicesStore } from "@/stores/services";
import type { ComponentInstance } from "vue";
import VirtualScroller from "./VirtualScroller.vue";
import type { ClientLogEntry, ScrollPosition } from "@/types/client";
import LogEntry from "./LogEntry.vue";
import LogViewerControls from "./LogViewerControls.vue";

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
    .map((log: any, index: number) => ({ ...log, elementIndex: index }))
    .filter(({ level }: any) => level === "ERR")
);

const errorsAbove = ref<typeof errors.value>([]);
const errorsBelow = ref<typeof errors.value>([]);

const searchQuery = ref("");

const commonBasePath = computed(() => {
  const fileRegex = /([/\\][\w\s\-.@/\\]+[/\\][\w\s\-.@/\\]+\.[\w]+)/g;
  const allPaths: string[] = [];

  // Extract all file paths from all logs
  service.value.logs.forEach((log: ClientLogEntry) => {
    const matches = [...log.message.matchAll(fileRegex)];
    matches.forEach((m) => allPaths.push(m[1].trim()));
  });

  if (!service.value.path || allPaths.length === 0) return "";

  // Generate candidate paths: project path and up to 2 levels up
  const parts = service.value.path.split("/");
  const candidates = [
    service.value.path, // Most specific
    parts.slice(0, -1).join("/"), // 1 level up
    parts.slice(0, -2).join("/"), // 2 levels up (most general)
  ].filter((p) => p.length > 0);

  // Check from most specific to most general
  // Use the most specific candidate that is an ancestor of OTHER file paths (not in project path)
  for (const candidate of candidates) {
    // Check if there are paths outside the project path that share this candidate as ancestor
    const pathsOutsideProject = allPaths.filter(
      (p) => !p.startsWith(service.value.path + "/") && p !== service.value.path
    );
    const hasSharedAncestor = pathsOutsideProject.some((p) =>
      p.startsWith(candidate + "/")
    );

    if (hasSharedAncestor) {
      return candidate;
    }
  }

  return service.value.path;
});

const updateErrorNavigation = () => {
  const range = virtualScroller.value?.getVisibleRange();
  if (!range) return;

  errorsAbove.value = errors.value.filter(
    (error: any) => error.elementIndex < range.start
  );

  errorsBelow.value = errors.value.filter(
    (error: any) => error.elementIndex > range.end
  );

  currentOrPreviousErrorIndex.value = errors.value.findIndex(
    (error: any) =>
      error.elementIndex <= range.end && error.elementIndex >= range.start
  );
};

const filteredLogs = computed(() => {
  if (!searchQuery.value) return service.value.logs;

  const query = searchQuery.value.toLowerCase();
  return service.value.logs.filter(
    (log: any) =>
      log.lines.some((line: string) => line.toLowerCase().includes(query)) ||
      log.level.toLowerCase().includes(query) ||
      log.timestamp.toLowerCase().includes(query)
  );
});

const handleScroll = ({
  scrollTop: _scrollTop,
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
    virtualScroller.value?.scrollToIndex(topIndex, offset);
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
    isScrolledToBottom.value
      ? undefined
      : {
          topIndex: range.start,
          offset,
        }
  );
});

const scrollToBottom = () => {
  virtualScroller.value?.scrollToBottom();
};

const navigateError = (error: (typeof errors.value)[number] | undefined) => {
  if (!error) return;
  virtualScroller.value?.scrollToIndex(error.elementIndex, 0);
};

const clearLogs = async () => {
  await store.clearLogs(props.serviceId);
  virtualScroller.value?.scrollToIndex(0, 0);
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
