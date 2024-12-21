<template>
  <div class="flex-1 flex flex-col overflow-hidden relative">
    <ServiceLogViewer
      v-for="service in services"
      :key="service.name"
      :service="service"
      :class="{
        'opacity-0 pointer-events-none': selectedService?.name !== service.name,
        'opacity-100': selectedService?.name === service.name
      }"
      class="absolute inset-0"
    />
  </div>
</template>

<script setup lang="ts">
import { useServicesStore } from '~/stores/services'
import { storeToRefs } from 'pinia';

const store = useServicesStore()
const { selectedService, services } = storeToRefs(store)
</script>