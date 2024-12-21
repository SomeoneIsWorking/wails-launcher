<template>
  <div class="w-64 bg-gray-100 overflow-y-auto flex flex-col h-full">
    <div class="flex-1 overflow-y-auto">
      <div
        v-for="service in services"
        :key="service.name"
        @click="store.selectService(service)"
        :class="[
          'p-4',
          selectedService === service ? 'bg-blue-300' : 'hover:bg-blue-200 cursor-pointer',
        ]"
      >
        <div class="flex items-center justify-between mb-2">
          <span>
            {{ service.name }}
          </span>
          <div class="flex items-center gap-2">
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
            @click="store.startService(service)"
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
            @click="store.stopService(service)"
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
    
    <div class="p-4 border-t">
      <button
        @click="showDialog = true"
        class="w-full px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
      >
        Add Service
      </button>
    </div>

    <!-- Dialog -->
    <div v-if="showDialog" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white p-6 rounded-lg shadow-lg w-96">
        <h3 class="text-lg font-medium mb-4">Add New Service</h3>
        <input
          v-model="servicePath"
          type="text"
          placeholder="Enter service path"
          class="w-full px-3 py-2 border rounded mb-4"
          @keyup.enter="addService"
        />
        <div class="flex justify-end gap-2">
          <button
            @click="showDialog = false"
            class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded"
          >
            Cancel
          </button>
          <button
            @click="addService"
            class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
          >
            Add
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useServicesStore } from '~/stores/services'
import { ref } from 'vue'

const store = useServicesStore()
const { services, selectedService } = storeToRefs(store)

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

const showDialog = ref(false)
const servicePath = ref('')

async function addService() {
  if (!servicePath.value) return
  
  try {
    await store.addService(servicePath.value)
    servicePath.value = '' // Clear input
    showDialog.value = false // Close dialog
  } catch (error) {
    console.error('Failed to add service:', error)
  }
}
</script>
