<template>
  <div
    @click="store.selectService(serviceId)"
    @contextmenu.prevent="showContextMenu"
    :class="[
      'p-4 pl-8',
      isSelected
        ? 'bg-blue-300'
        : 'hover:bg-blue-200 cursor-pointer',
    ]"
  >
    <div class="flex items-center justify-between mb-2">
      <span class="text-gray-800 mr-4">
        {{ service.name }}
      </span>
      <div class="flex items-center gap-2">
        <button
          @click.stop="$emit('edit')"
          class="text-gray-500 hover:text-gray-700"
        >
          <SettingsIcon :size="16" />
        </button>
        <div
          :class="['w-3 h-3 rounded-full', statusColor(service.status)]"
        ></div>
        <span
          v-if="store.getUnreadErrorCount(serviceId)"
          class="bg-red-500 text-white text-xs px-2 rounded-full"
        >
          {{ store.getUnreadErrorCount(serviceId) }}
        </span>
      </div>
    </div>

    <a
      v-if="service.url"
      :href="service.url"
      target="_blank"
      class="text-xs text-blue-600 hover:underline mb-2"
    >
      {{ formatUrl(service.url) }}
    </a>

    <div class="flex gap-2">
      <button
        @click.stop="store.startService(serviceId)"
        :disabled="
          service.status === 'running' ||
          service.status === 'starting' ||
          service.status === 'stopping' ||
          service.status === 'initializing'
        "
        class="p-2 text-sm bg-green-500 text-white rounded disabled:opacity-50 hover:bg-green-600"
        title="Start"
      >
        <PlayIcon :size="16" />
      </button>
      <button
        @click.stop="store.startServiceWithoutBuild(serviceId)"
        :disabled="
          service.status === 'running' ||
          service.status === 'starting' ||
          service.status === 'stopping' ||
          service.status === 'initializing'
        "
        class="p-2 text-sm bg-purple-500 text-white rounded disabled:opacity-50 hover:bg-purple-600"
        title="Run without building"
      >
        <ZapIcon :size="16" />
      </button>
      <button
        @click.stop="store.stopService(serviceId)"
        :disabled="
          service.status === 'stopped' ||
          service.status === 'starting' ||
          service.status === 'stopping'
        "
        class="p-2 text-sm bg-red-500 text-white rounded disabled:opacity-50 hover:bg-red-600"
        title="Stop"
      >
        <SquareIcon :size="16" />
      </button>
      <button
        @click.stop="store.restartService(serviceId)"
        :disabled="service.status !== 'running'"
        class="p-2 text-sm bg-blue-500 text-white rounded disabled:opacity-50 hover:bg-blue-600"
        title="Restart"
      >
        <RotateCwIcon :size="16" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useServicesStore } from "@/stores/services";
import { useContextMenuStore } from "@/stores/contextMenu";
import { confirmDialog } from "@/stores/confirm";
import { ClientServiceInfo } from "@/types/client";
import {
  SettingsIcon,
  PlayIcon,
  SquareIcon,
  RotateCwIcon,
  ZapIcon,
} from "lucide-vue-next";

interface Props {
  serviceId: string;
  service: ClientServiceInfo;
  isSelected: boolean;
}

const props = defineProps<Props>();

defineEmits<{
  edit: [];
}>();

const store = useServicesStore();
const contextMenuStore = useContextMenuStore();

function showContextMenu(event: MouseEvent) {
  contextMenuStore.show(event, [
    {
      label: "Delete Service",
      action: async () => {
        const confirmed = await confirmDialog({
          title: "Delete Service",
          message: `Are you sure you want to delete "${props.service.name}"?`,
          confirmText: "Delete",
          cancelText: "Cancel",
          confirmVariant: "danger"
        });
        
        if (confirmed) {
          await store.deleteService(props.serviceId);
        }
        contextMenuStore.hide();
      },
      disabled: props.service.status !== 'stopped',
    },
  ]);
}

const statusColor = (status: string) =>
  ({
    running: "bg-green-500",
    stopped: "bg-gray-500",
    error: "bg-red-500",
    starting: "bg-yellow-500",
    stopping: "bg-yellow-500",
    initializing: "bg-yellow-500",
  }[status]);

function formatUrl(url: string): string {
  return url.replace(/^https?:\/\//, "");
}
</script>
