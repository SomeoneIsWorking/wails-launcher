<template>
  <VDialog
    v-if="confirmStore.visible"
    :title="confirmStore.title"
    @close="confirmStore.cancel()"
  >
    <div class="text-gray-700">
      {{ confirmStore.message }}
    </div>

    <template #footer>
      <button
        @click="confirmStore.cancel()"
        class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
      >
        {{ confirmStore.cancelText }}
      </button>
      <button
        @click="confirmStore.confirm()"
        :class="[
          'px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors',
          confirmStore.confirmVariant === 'danger'
            ? 'bg-red-600 hover:bg-red-700'
            : 'bg-blue-600 hover:bg-blue-700'
        ]"
      >
        {{ confirmStore.confirmText }}
      </button>
    </template>
  </VDialog>
</template>

<script setup lang="ts">
import { useConfirmStore } from "@/stores/confirm";
import VDialog from "./VDialog.vue";

const confirmStore = useConfirmStore();
</script>
