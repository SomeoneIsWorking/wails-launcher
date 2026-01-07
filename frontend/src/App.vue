<template>
  <div class="flex h-screen">
    <ServiceList />
    <LogViewer />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useServicesStore } from "./stores/services";
import ServiceList from "./components/ServiceList.vue";
import LogViewer from "./components/LogViewer.vue";

const store = useServicesStore();

// Load services on mount
onMounted(async () => {
  if (Object.entries(store.services).length === 0) {
    await store.loadAll();
    if (Object.keys(store.services).length > 0) {
      store.selectService(Object.keys(store.services)[0]);
    }
  }
});
</script>
