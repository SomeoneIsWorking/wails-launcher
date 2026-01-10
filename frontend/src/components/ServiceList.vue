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
        <ServiceItem
          v-for="(service, serviceId) in group.services"
          :key="serviceId"
          :service-id="serviceId"
          :service="service"
          :is-selected="selectedService === service"
          @edit="editService(serviceId)"
        />
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
          @click="openImportDialog"
          class="flex flex-col items-center justify-center p-2 bg-gray-50 border border-gray-200 text-gray-600 rounded-lg hover:bg-indigo-50 hover:text-indigo-700 hover:border-indigo-200 transition-colors"
          title="Import Project"
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
    <ServiceConfig
      v-if="editingServiceId"
      :service-id="editingServiceId"
      @close="editingServiceId = undefined"
    />

    <!-- Group Config Dialog -->
    <GroupConfig
      v-if="editingGroupId"
      :group-id="editingGroupId"
      @close="editingGroupId = undefined"
    />

    <!-- Import Dialog -->
    <ImportDialog
      v-if="importDialog"
      @close="importDialog = false"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useServicesStore } from "@/stores/services";
import { useContextMenuStore } from "@/stores/contextMenu";
import { ref } from "vue";
import {
  SettingsIcon,
  RefreshCwIcon,
  PlusIcon,
  FolderPlusIcon,
  DownloadIcon,
} from "lucide-vue-next";
import ServiceConfig from "./ServiceConfig.vue";
import GroupConfig from "./GroupConfig.vue";
import ImportDialog from "./ImportDialog.vue";
import ServiceItem from "./ServiceItem.vue";

const store = useServicesStore();
const contextMenuStore = useContextMenuStore();
const { groups, selectedService } = storeToRefs(store);

const editingServiceId = ref<string>();
const editingGroupId = ref<string>();
const importDialog = ref(false);

function openImportDialog() {
  importDialog.value = true;
}

function editService(id: string) {
  editingServiceId.value = id;
}

function editGroup(id: string) {
  editingGroupId.value = id;
}

function showGroupContextMenu(event: MouseEvent, groupId: string) {
  contextMenuStore.show(event, [
    {
      label: "Launch Group",
      action: () => {
        store.startGroup(groupId);
        contextMenuStore.hide();
      },
    },
  ]);
}
</script>
