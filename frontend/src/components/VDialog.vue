<template>
  <Transition
    enter-active-class="transition duration-200 ease-out"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
    appear
  >
    <div
      class="fixed inset-0 bg-slate-900/60 flex items-center justify-center z-50 p-4"
      @click.self="$emit('close')"
    >
      <div
        class="bg-white rounded-2xl shadow-2xl flex flex-col border border-gray-100 max-h-[80vh] max-w-[80vw]"
      >
        <!-- Header -->
        <div
          class="px-6 py-4 border-b border-gray-100 flex items-center justify-between flex-shrink-0"
        >
          <h3 class="text-lg font-bold text-gray-900 flex items-center gap-2">
            <slot name="title">
              {{ title }}
            </slot>
          </h3>
          <button
            @click="$emit('close')"
            class="text-gray-400 hover:text-gray-600 transition-colors p-1 hover:bg-gray-100 rounded-lg"
          >
            <XIcon :size="20" />
          </button>
        </div>

        <!-- Scrollable content -->
        <div class="p-6 overflow-y-auto min-h-0 flex-1">
          <slot />
        </div>

        <!-- Footer -->
        <div
          v-if="$slots.footer"
          class="px-6 py-4 border-t border-gray-100 flex gap-3 flex-shrink-0"
        >
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { XIcon } from "lucide-vue-next";

interface Props {
  title?: string;
}

withDefaults(defineProps<Props>(), {});

defineEmits<{
  close: [];
}>();
</script>
