<template>
  <div class="flex h-screen">
    <ServiceList />
    <LogViewer />
  </div>
</template>

<script setup lang="ts">
import type { ServiceInfo } from "./server/types";
import { useServicesStore } from "./stores/services";

const store = useServicesStore();

if (store.services.length === 0) {
  const services = await $fetch<ServiceInfo[]>("/api/services");
  store.addServices(services);
}
store.selectService(store.services[0]);
</script>
