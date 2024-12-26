<template>
  <div class="w-64 bg-gray-100 overflow-y-auto flex flex-col h-full">
    <div class="flex-1 overflow-y-auto">
      <div
        v-for="(service, id) in services"
        :key="id"
        @click="store.selectService(id)"
        :class="[
          'p-4',
          selectedService === service
            ? 'bg-blue-300'
            : 'hover:bg-blue-200 cursor-pointer',
        ]"
      >
        <div class="flex items-center justify-between mb-2">
          <span>
            {{ service.name }}
          </span>
          <div class="flex items-center gap-2">
            <button
              @click.stop="editService(id)"
              class="text-gray-500 hover:text-gray-700"
            >
              <SettingsIcon :size="16" />
            </button>
            <div
              :class="['w-3 h-3 rounded-full', statusColor(service.status)]"
            ></div>
            <span
              v-if="store.getUnreadErrorCount(service.name)"
              class="bg-red-500 text-white text-xs px-2 rounded-full"
            >
              {{ store.getUnreadErrorCount(service.name) }}
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
            @click="store.startService(id)"
            :disabled="
              service.status === 'running' ||
              service.status === 'starting' ||
              service.status === 'stopping'
            "
            class="px-2 py-1 text-sm bg-green-500 text-white rounded disabled:opacity-50 min-w-[60px]"
          >
            {{ service.status === "starting" ? "..." : "Start" }}
          </button>
          <button
            @click="store.stopService(id)"
            :disabled="
              service.status === 'stopped' ||
              service.status === 'starting' ||
              service.status === 'stopping'
            "
            class="px-2 py-1 text-sm bg-red-500 text-white rounded disabled:opacity-50 min-w-[60px]"
          >
            {{ service.status === "stopping" ? "..." : "Stop" }}
          </button>
        </div>
      </div>
    </div>

    <div class="p-4 border-t flex gap-2">
      <button
        @click="editingServiceId = 'new'"
        class="flex-1 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
      >
        Add Service
      </button>
      <button
        @click="store.reloadConfig"
        class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
        title="Reload config from services.json"
      >
        <RefreshCwIcon :size="16" />
      </button>
    </div>

    <!-- Config Dialog -->
    <div
      v-if="editingServiceId"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <ServiceConfig
        :service-id="editingServiceId"
        @close="editingServiceId = undefined"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useServicesStore } from "~/stores/services";
import { ref } from "vue";
import { SettingsIcon, RefreshCwIcon } from "lucide-vue-next";

const store = useServicesStore();
const { services, selectedService } = storeToRefs(store);

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

const editingServiceId = ref<string>();

function editService(id: string) {
  editingServiceId.value = id;
}
</script>
