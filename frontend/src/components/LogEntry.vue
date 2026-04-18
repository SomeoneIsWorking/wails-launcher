<template>
  <VIntersectionObserver
    :enabled="
      log.level === 'ERR' &&
      !log.read &&
      serviceName === store.selectedService?.name
    "
    @intersection="markLogAsRead"
  >
    <!-- Raw mode: plain text, no link processing -->
    <div v-if="showRaw" :class="[logLevelClass, wrap ? 'border-b border-gray-700/20' : '']" class="opacity-75">
      {{ log.raw }}
    </div>

    <!-- Parsed mode: URL / file-path segments highlighted -->
    <div v-else :class="[logLevelClass, wrap ? 'border-b border-gray-700/20' : '']">
      <div v-for="(segmentItems, index) in processedContent" :key="index">
        <template v-for="segment in segmentItems">
          <a
            v-if="segment.type === 'url'"
            :href="segment.url"
            target="_blank"
            class="px-1 rounded bg-blue-900/30 text-blue-300 hover:bg-blue-800/50 hover:text-blue-200"
            >{{ segment.text }}</a
          >
          <a
            v-else-if="segment.type === 'file'"
            :href="segment.url"
            class="px-1 rounded bg-purple-900/30 text-purple-300 hover:bg-purple-800/50 hover:text-purple-200"
            >{{ segment.text }}</a
          >
          <span v-else>{{ segment.text }}</span>
        </template>
      </div>
    </div>
  </VIntersectionObserver>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useServicesStore } from "@/stores/services";
import type { ClientLogEntry } from "@/types/client";
import VIntersectionObserver from "./VIntersectionObserver.vue";

const props = defineProps<{
  log: ClientLogEntry;
  serviceName: string;
  servicePath?: string;
  commonBasePath?: string;
  showRaw?: boolean;
  wrap?: boolean;
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

type ContentSegment = {
  type: "text" | "url" | "file";
  text: string;
  url?: string;
};

const processedContent = computed(() =>
  props.log.lines.map((line) => processLogContent(line))
);
const store = useServicesStore();

const markLogAsRead = async () => {
  store.markLogAsRead(props.serviceName, props.log.timestamp);
  props.log.read = true;
};

const createVscodeUrl = (filePath: string, lineNumber?: string) => {
  const line = lineNumber ? parseInt(lineNumber) : 1;
  return `vscode://file/${filePath}${line ? `:${line}` : ""}`;
};

const processLogContent = (content: string): ContentSegment[] => {
  const segments: ContentSegment[] = [];
  const urlRegex = /(https?:\/\/[^\s]+)/g;

  // Matches both absolute paths (/foo/bar/File.cs) and relative paths
  // (Namespace.Name/Dir/File.cs), followed by an optional location suffix in
  // either `:line N` or ` (N,M)` format.
  //
  // Group 1 — file path
  // Group 2 — line number from `:line N`
  // Group 3 — line number from `(N,M)`
  const fileRegex =
    /((?:[/\\]|(?=[\w][\w.\-@]*[/\\]))[\w\s\-.@/\\]+[/\\][\w\s\-.@]+\.[\w]{1,6})(?::line (\d+)|\s*\((\d+),\d+\))?/g;

  let lastIndex = 0;
  const matches: Array<{
    index: number;
    length: number;
    segment: ContentSegment;
  }> = [];

  // Find all URL matches
  let match;
  while ((match = urlRegex.exec(content)) !== null) {
    matches.push({
      index: match.index,
      length: match[0].length,
      segment: {
        type: "url",
        text: match[1],
        url: match[1],
      },
    });
  }

  // Find all file path matches
  while ((match = fileRegex.exec(content)) !== null) {
    const filePath = match[1];
    // Accept line number from either `:line N` (group 2) or `(N,M)` (group 3)
    const lineNumber = match[2] ?? match[3];
    const trimmedPath = filePath.trim();

    // Resolve relative paths against commonBasePath so VS Code can open them
    const absPath =
      trimmedPath.startsWith("/") || /^[A-Za-z]:/.test(trimmedPath)
        ? trimmedPath
        : props.commonBasePath
          ? `${props.commonBasePath}/${trimmedPath}`
          : trimmedPath;

    let displayPath = match[0];
    if (props.commonBasePath && absPath.startsWith(props.commonBasePath)) {
      const relativePath = absPath
        .substring(props.commonBasePath.length)
        .replace(/^\//, "");
      displayPath = lineNumber
        ? `${relativePath}:${lineNumber}`
        : relativePath;
    }

    matches.push({
      index: match.index,
      length: match[0].length,
      segment: {
        type: "file",
        text: displayPath,
        url: createVscodeUrl(absPath, lineNumber),
      },
    });
  }

  // Sort matches by index
  matches.sort((a, b) => a.index - b.index);

  // Build segments array
  for (const match of matches) {
    // Add text before this match
    if (match.index > lastIndex) {
      const text = content.substring(lastIndex, match.index);
      if (text) {
        segments.push({ type: "text", text });
      }
    }
    segments.push(match.segment);
    lastIndex = match.index + match.length;
  }

  // Add remaining text
  if (lastIndex < content.length) {
    segments.push({ type: "text", text: content.substring(lastIndex) });
  }

  return segments.length > 0 ? segments : [{ type: "text", text: content }];
};
</script>
