<template>
  <VIntersectionObserver
    v-if="
      log.level === 'ERR' &&
      !log.read &&
      serviceName === store.selectedService?.name
    "
    @intersection="markLogAsRead"
  >
    <div
      :class="[
        'whitespace-pre-wrap leading-5 py-0.5 flex gap-2',
        logLevelClass,
        { 'opacity-75': log.read && log.level === 'ERR' },
      ]"
    >
      <span v-if="log.stream === 'stderr'" class="flex-none opacity-50 text-[10px] uppercase border border-current px-1 rounded h-fit mt-1">stderr</span>
      <div v-html="processedContent" class="flex-1" />
    </div>
  </VIntersectionObserver>
  <div
    v-else
    :class="[
      'whitespace-pre-wrap leading-5 py-0.5 break-all overflow-wrap-anywhere flex gap-2',
      logLevelClass
    ]"
  >
    <span v-if="log.stream === 'stderr'" class="flex-none opacity-50 text-[10px] uppercase border border-current px-1 rounded h-fit mt-1">stderr</span>
    <div v-html="processedContent" class="flex-1" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useServicesStore } from "@/stores/services";
import type { ClientLogEntry } from "@/types/client";
import VIntersectionObserver from './VIntersectionObserver.vue';

const props = defineProps<{
  log: ClientLogEntry;
  serviceName: string;
}>();

const logLevelClass = computed(
  () =>
    ({
      ERR: "text-red-400 font-medium",
      WARN: "text-yellow-400",
      INF: "text-blue-400",
      DBG: "text-gray-400",
    }[props.log.level])
);

const processedContent = computed(() => processLogContent(props.log.raw));
const store = useServicesStore();

const markLogAsRead = async () => {
  store.markLogAsRead(props.serviceName, props.log.timestamp);
  props.log.read = true;
};

const createVscodeUrl = (filePath: string, lineNumber?: string) => {
  const line = lineNumber ? parseInt(lineNumber) : 1;
  return `vscode://file/${filePath}${line ? `:${line}` : ""}`;
};

const processLogContent = (content: string) => {
  // First process URLs
  const urlRegex = /(https?:\/\/[^\s]+)/g;
  content = content.replace(
    urlRegex,
    (url) =>
      `<a href="${url}" target="_blank" class="px-1 rounded bg-blue-900/30 text-blue-300 hover:bg-blue-800/50 hover:text-blue-200">${url}</a>`
  );

  // Then process file paths
  const fileRegex =
    /([\/\\][\w\s\-./\\]+[\/\\][\w\s\-./\\]+\.[\w]+)(?::line (\d+))?/g;
  return content.replace(fileRegex, (match, filePath, lineNumber) => {
    const trimmedPath = filePath.trim();
    return `<a href="${createVscodeUrl(
      trimmedPath,
      lineNumber
    )}" class="px-1 rounded bg-purple-900/30 text-purple-300 hover:bg-purple-800/50 hover:text-purple-200">${match}</a>`;
  });
};
</script>

<style scoped>
.overflow-wrap-anywhere {
  overflow-wrap: anywhere;
  word-break: break-word;
}
</style>
