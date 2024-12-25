<template>
  <div class="flex h-screen">
    <ServiceList />
    <LogViewer />
  </div>
</template>

<script setup lang="ts">
import { useServicesStore } from "./stores/services";

const store = useServicesStore();

if (Object.entries(store.services).length === 0) {
  const services = await $fetch("/api/services");
  store.addServices(services);
  if (Object.keys(services).length > 0) {
    store.selectService(Object.keys(services)[0]);
  }
}
</script>
