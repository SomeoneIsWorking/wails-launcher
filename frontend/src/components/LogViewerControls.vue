<template>
  <div
    class="opacity-25 transition-opacity hover:opacity-100 absolute z-10 top-4 right-6 flex items-stretch bg-gray-800/90 rounded-full shadow-lg backdrop-blur-sm text-sm text-gray-300"
  >
    <div class="px-5 flex items-center">
      <input
        :value="searchQuery"
        @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        type="text"
        placeholder="Search logs..."
        class="bg-transparent border-none outline-none text-white placeholder-gray-400 w-48"
      />
    </div>
    <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
    <button @click="$emit('clearLogs')" class="px-5 py-2 hover:text-white">
      Clear Logs
    </button>
    <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
    <div class="px-5 flex items-center">
      Error {{ currentOrPreviousErrorIndex + 1 }} of {{ errors.length }}
    </div>
    <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
    <div class="flex items-center gap-2 px-5 py-2">
      <button
        @click="$emit('navigateError', errorsAbove.at(-1))"
        :disabled="!errorsAbove.length"
        class="nav-buttons"
      >
        {{ errorsAbove.length }}
        <ChevronUp class="w-5 h-5" />
      </button>
      <button
        @click="$emit('navigateError', errorsBelow[0])"
        :disabled="!errorsBelow.length"
        class="nav-buttons"
      >
        {{ errorsBelow.length }}
        <ChevronDown class="w-5 h-5" />
      </button>
    </div>
    <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
    <!-- Wrap toggle: WrapText = wrapping ON, ArrowRightToLine = wrapping OFF -->
    <button
      @click="$emit('update:wrap', !wrap)"
      :title="wrap ? 'Wrapping on — click to disable' : 'Wrapping off — click to enable'"
      class="px-3 py-2 hover:text-white transition-colors"
      :class="wrap ? 'text-blue-400' : 'text-gray-500'"
    >
      <WrapText v-if="wrap" class="w-4 h-4" />
      <ArrowRightToLine v-else class="w-4 h-4" />
    </button>
    <span class="inline-block w-0.5 flex-grow bg-gray-500"></span>
    <!-- Raw / parsed toggle: Code = raw mode, FileText = parsed mode -->
    <button
      @click="$emit('update:showRaw', !showRaw)"
      :title="showRaw ? 'Raw mode — click for parsed' : 'Parsed mode — click for raw'"
      class="px-3 py-2 hover:text-white transition-colors rounded-r-full"
      :class="showRaw ? 'text-blue-400' : 'text-gray-500'"
    >
      <Code v-if="showRaw" class="w-4 h-4" />
      <FileText v-else class="w-4 h-4" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ChevronUp, ChevronDown, WrapText, ArrowRightToLine, Code, FileText } from "lucide-vue-next";

defineProps<{
  searchQuery: string;
  errors: any[];
  errorsAbove: any[];
  errorsBelow: any[];
  currentOrPreviousErrorIndex: number;
  wrap: boolean;
  showRaw: boolean;
}>();

defineEmits<{
  'update:searchQuery': [value: string];
  'update:wrap': [value: boolean];
  'update:showRaw': [value: boolean];
  clearLogs: [];
  navigateError: [error: any];
}>();
</script>
