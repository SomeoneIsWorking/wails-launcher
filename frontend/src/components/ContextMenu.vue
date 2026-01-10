<template>
  <div
    v-if="contextMenuStore.visible"
    class="fixed bg-white border border-gray-300 rounded shadow-lg z-50"
    :style="{ left: contextMenuStore.x + 'px', top: contextMenuStore.y + 'px' }"
    @click.stop
  >
    <div
      v-for="option in contextMenuStore.options"
      :key="option.label"
      @click="!option.disabled && option.action()"
      class="px-4 py-2"
      :class="option.disabled ? 'text-gray-400 cursor-not-allowed' : 'hover:bg-gray-100 cursor-pointer'"
    >
      {{ option.label }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useContextMenuStore } from "@/stores/contextMenu";

const contextMenuStore = useContextMenuStore();

// Hide context menu on click outside
document.addEventListener('click', () => contextMenuStore.hide());
</script>
