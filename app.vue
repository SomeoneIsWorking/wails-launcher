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
}
store.selectService(store.services[0]);
</script>
