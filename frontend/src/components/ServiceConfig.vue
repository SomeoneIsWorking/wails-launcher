<template>
  <div class="bg-white p-6 rounded-lg shadow-lg w-auto min-w-[30rem]">
    <h3 class="text-lg font-medium mb-4">
      {{ serviceId === "new" ? "Add" : "Edit" }} Service
    </h3>

    <div class="space-y-4">
      <div v-if="serviceId === 'new'">
        <label class="block text-sm font-medium mb-1"> Group </label>
        <select
          v-model="selectedGroupId"
          class="w-full px-3 py-2 border rounded"
        >
          <option value="">Select a group</option>
          <option
            v-for="(group, id) in store.groups"
            :key="id"
            :value="id"
          >
            {{ group.name }}
          </option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium mb-1"> Name </label>
        <input
          v-model="form.name"
          type="text"
          class="w-full px-3 py-2 border rounded"
        />
      </div>

      <div>
        <label class="block text-sm font-medium mb-1"> Path </label>
        <input
          v-model="form.path"
          type="text"
          class="w-full px-3 py-2 border rounded"
        />
      </div>

      <EnvVariables v-model="form.env" />
    </div>

    <div class="flex justify-end gap-2 mt-6">
      <button
        @click="$emit('close')"
        class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded"
      >
        Cancel
      </button>
      <button
        @click="save"
        class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
      >
        Save
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { map } from "lodash-es";
import { useServicesStore } from "@/stores/services";
import EnvVariables from "./EnvVariables.vue";

const store = useServicesStore();

const props = defineProps<{
  serviceId: string;
}>();

const emit = defineEmits<{
  close: [];
}>();
const form = ref(setup());
const selectedGroupId = ref("");

function setup() {
  const value = props.serviceId;
  if (value === "new") {
    return { name: "", path: "", env: [] };
  }
  const service = store.services[value];
  if (!service) {
    throw new Error(`Service with id ${value} not found`);
  }
  return {
    name: service.name,
    path: service.path,
    env: map(Object.entries(service.env), ([key, value], index) => ({
      index,
      key,
      value,
    })),
  };
}

function toModel() {
  return {
    name: form.value.name,
    path: form.value.path,
    env: Object.fromEntries(
      form.value.env.map(({ key, value }) => [key, value])
    ),
  };
}
async function save() {
  if (!props.serviceId) {
    throw new Error("serviceId is required");
  }
  try {
    if (props.serviceId === "new") {
      if (selectedGroupId.value) {
        await store.addServiceToGroup(selectedGroupId.value, toModel());
      } else {
        await store.addService(toModel());
      }
    } else {
      // For editing, find the group containing this service
      const groupId = Object.keys(store.groups).find(id =>
        Object.keys(store.groups[id].services).includes(props.serviceId)
      );
      if (groupId) {
        await store.updateServiceInGroup(groupId, props.serviceId, toModel());
      } else {
        await store.updateService(props.serviceId, toModel());
      }
    }
    emit("close");
  } catch (error) {
    console.error("Failed to save service:", error);
  }
}
</script>
