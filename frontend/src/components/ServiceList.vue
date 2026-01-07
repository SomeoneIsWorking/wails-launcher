<template>
  <div class="bg-gray-100 overflow-y-auto flex flex-col h-full">
    <div class="flex-1 overflow-y-auto">
      <div
        v-for="(group, groupId) in groups"
        :key="groupId"
        class="mb-4"
      >
        <div class="px-4 py-2 bg-gray-200 font-semibold text-gray-800 flex items-center justify-between"
             @contextmenu.prevent="showGroupContextMenu($event, groupId)">
          <span>{{ group.name }}</span>
          <button
            @click.stop="editGroup(groupId)"
            class="text-gray-500 hover:text-gray-700"
          >
            <SettingsIcon :size="16" />
          </button>
        </div>
        <div
          v-for="(service, serviceId) in group.services"
          :key="serviceId"
          @click="store.selectService(serviceId)"
          @contextmenu.prevent="showServiceContextMenu($event, serviceId, service)"
          :class="[
            'p-4 pl-8',
            selectedService === service
              ? 'bg-blue-300'
              : 'hover:bg-blue-200 cursor-pointer',
          ]"
        >
          <div class="flex items-center justify-between mb-2">
            <span class="text-gray-800">
              {{ service.name }}
            </span>
            <div class="flex items-center gap-2">
              <button
                @click.stop="editService(serviceId)"
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
              @click="store.startService(serviceId)"
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
              @click="store.stopService(serviceId)"
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
              @click="store.restartService(serviceId)"
              :disabled="service.status !== 'running'"
              class="p-2 text-sm bg-blue-500 text-white rounded disabled:opacity-50 hover:bg-blue-600"
              title="Restart"
            >
              <RotateCwIcon :size="16" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="p-4 border-t flex gap-2 flex-wrap">
      <button
        @click="editingServiceId = 'new'"
        class="flex-1 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
      >
        Add Service
      </button>
      <button
        @click="editingGroupId = 'new'"
        class="flex-1 px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"
      >
        Add Group
      </button>
      <button
        @click="importSLNDialog = true"
        class="flex-1 px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600"
      >
        Import SLN
      </button>
      <button
        @click="store.reloadConfig"
        class="flex-1 px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
        title="Reload config from services.json"
      >
        <RefreshCwIcon :size="16" />
      </button>
    </div>

    <!-- Service Config Dialog -->
    <div
      v-if="editingServiceId"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <ServiceConfig
        :service-id="editingServiceId"
        @close="editingServiceId = undefined"
      />
    </div>

    <!-- Group Config Dialog -->
    <div
      v-if="editingGroupId"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <GroupConfig
        :group-id="editingGroupId"
        @close="editingGroupId = undefined"
      />
    </div>

    <!-- Import SLN Dialog -->
    <div
      v-if="importSLNDialog"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <div class="bg-white p-6 rounded-lg w-96">
        <h3 class="text-lg font-semibold mb-4">Import SLN File</h3>
        <input
          v-model="slnPath"
          type="text"
          placeholder="Path to .sln file"
          class="w-full px-3 py-2 border rounded mb-4"
        />
        <div class="flex gap-2">
          <button
            @click="importSLN"
            class="flex-1 px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600"
          >
            Import
          </button>
          <button
            @click="importSLNDialog = false"
            class="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>

    <!-- Context Menu -->
    <div
      v-if="contextMenu.visible"
      class="fixed bg-white border border-gray-300 rounded shadow-lg z-50"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      @click.stop
    >
      <div
        v-for="option in contextMenu.options"
        :key="option.label"
        @click="option.action"
        class="px-4 py-2 hover:bg-gray-100 cursor-pointer"
        :class="{ 'text-gray-400 cursor-not-allowed': option.disabled }"
      >
        {{ option.label }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useServicesStore } from "@/stores/services";
import { ref } from "vue";
import { SettingsIcon, RefreshCwIcon, PlayIcon, SquareIcon, RotateCwIcon } from "lucide-vue-next";
import ServiceConfig from "./ServiceConfig.vue";
import GroupConfig from "./GroupConfig.vue";

const store = useServicesStore();
const { groups, selectedService } = storeToRefs(store);

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
const editingGroupId = ref<string>();
const importSLNDialog = ref(false);
const slnPath = ref("");

const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  options: [] as { label: string; action: () => void; disabled?: boolean }[],
});

function editService(id: string) {
  editingServiceId.value = id;
}

function editGroup(id: string) {
  editingGroupId.value = id;
}

async function importSLN() {
  if (slnPath.value) {
    try {
      await store.importSLN(slnPath.value);
      importSLNDialog.value = false;
      slnPath.value = "";
    } catch (error) {
      console.error("Failed to import SLN:", error);
    }
  }
}

function showGroupContextMenu(event: MouseEvent, groupId: string) {
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    options: [
      {
        label: "Launch Group",
        action: () => {
          store.startGroup(groupId);
          hideContextMenu();
        },
      },
    ],
  };
}

function showServiceContextMenu(event: MouseEvent, serviceId: string, service: any) {
  const options = [];
  if (service.status === 'stopped') {
    options.push({
      label: "Delete Service",
      action: async () => {
        await store.deleteService(serviceId);
        hideContextMenu();
      },
    });
  }
  if (options.length > 0) {
    contextMenu.value = {
      visible: true,
      x: event.clientX,
      y: event.clientY,
      options,
    };
  }
}

function hideContextMenu() {
  contextMenu.value.visible = false;
}

// Hide context menu on click outside
document.addEventListener('click', hideContextMenu);
</script>
