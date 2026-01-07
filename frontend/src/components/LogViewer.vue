<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <ServiceLogViewer
      v-if="selectedService && selectedServiceId"
      :key="selectedService.path"
      :service-id="selectedServiceId"
    />
  </div>
</template>

<script setup lang="ts">
import { useServicesStore } from "@/stores/services";
import { storeToRefs } from "pinia";
import ServiceLogViewer from "./ServiceLogViewer.vue";

const store = useServicesStore();
const { selectedService, selectedServiceId } = storeToRefs(store);
</script>
