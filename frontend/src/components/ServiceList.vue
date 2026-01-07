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

    <div class="p-4 border-t bg-white space-y-2">
      <button
        @click="editingServiceId = 'new'"
        class="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-blue-600 text-white text-sm font-semibold rounded-lg hover:bg-blue-700 transition-all shadow-sm active:scale-95"
      >
        <PlusIcon :size="18" />
        New Service
      </button>
      <div class="grid grid-cols-3 gap-2">
        <button
          @click="editingGroupId = 'new'"
          class="flex flex-col items-center justify-center p-2 bg-gray-50 border border-gray-200 text-gray-600 rounded-lg hover:bg-emerald-50 hover:text-emerald-700 hover:border-emerald-200 transition-colors"
          title="Add Group"
        >
          <FolderPlusIcon :size="18" />
          <span class="text-[10px] mt-1 font-medium uppercase">Group</span>
        </button>
        <button
          @click="importSLNDialog = true"
          class="flex flex-col items-center justify-center p-2 bg-gray-50 border border-gray-200 text-gray-600 rounded-lg hover:bg-indigo-50 hover:text-indigo-700 hover:border-indigo-200 transition-colors"
          title="Import SLN"
        >
          <DownloadIcon :size="18" />
          <span class="text-[10px] mt-1 font-medium uppercase">Import</span>
        </button>
        <button
          @click="store.reloadConfig"
          class="flex flex-col items-center justify-center p-2 bg-gray-50 border border-gray-200 text-gray-600 rounded-lg hover:bg-orange-50 hover:text-orange-700 hover:border-orange-200 transition-colors"
          title="Reload config from services.json"
        >
          <RefreshCwIcon :size="18" />
          <span class="text-[10px] mt-1 font-medium uppercase">Reload</span>
        </button>
      </div>
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
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="importSLNDialog"
        class="fixed inset-0 bg-slate-900/60 flex items-center justify-center z-50 p-4"
        @click.self="importSLNDialog = false"
      >
        <div class="bg-white rounded-xl shadow-2xl w-full max-w-md overflow-hidden">
          <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between bg-gray-50/50">
          <h3 class="text-lg font-bold text-gray-800 flex items-center gap-2">
            <DownloadIcon :size="20" class="text-indigo-600" />
            Import SLN File
          </h3>
          <button @click="importSLNDialog = false" class="text-gray-400 hover:text-gray-600 transition-colors">
            <XIcon :size="20" />
          </button>
        </div>
        
        <div class="p-6">
          <p class="text-sm text-gray-500 mb-4">
            Select a Visual Studio Solution (.sln) file to automatically import projects as services.
          </p>
          
          <div class="space-y-4">
            <div>
              <label class="block text-xs font-bold text-gray-400 uppercase tracking-widest mb-1.5 ml-1">
                Solution Path
              </label>
              <div class="flex gap-2">
                <div class="relative flex-1">
                  <input
                    v-model="slnPath"
                    type="text"
                    placeholder="/path/to/solution.sln"
                    class="w-full pl-3 pr-3 py-2.5 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all"
                  />
                </div>
                <button
                  @click="browseSLN"
                  class="px-4 py-2 bg-white border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 hover:border-gray-300 transition-all flex items-center gap-2 shadow-sm whitespace-nowrap"
                >
                  <FolderOpenIcon :size="18" />
                  <span class="text-sm font-medium">Browse</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="px-6 py-4 bg-gray-50 border-t border-gray-100 flex gap-3">
          <button
            @click="importSLNDialog = false"
            class="flex-1 px-4 py-2.5 text-sm font-semibold text-gray-600 hover:text-gray-800 transition-colors"
          >
            Cancel
          </button>
          <button
            @click="importSLN"
            :disabled="!slnPath"
            class="flex-[2] px-4 py-2.5 bg-indigo-600 text-white text-sm font-semibold rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md shadow-indigo-200 active:scale-[0.98]"
          >
            Import Projects
          </button>
        </div>
      </div>
    </div>
    </Transition>

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
import {
  SettingsIcon,
  RefreshCwIcon,
  PlayIcon,
  SquareIcon,
  RotateCwIcon,
  PlusIcon,
  FolderPlusIcon,
  DownloadIcon,
  FolderOpenIcon,
  XIcon,
} from "lucide-vue-next";
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

async function browseSLN() {
  const path = await store.browse();
  if (path) {
    slnPath.value = path;
  }
}

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
