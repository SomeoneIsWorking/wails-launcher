<template>
  <VDialog
    :title="serviceId === 'new' ? 'Add Service' : 'Edit Service'"
    @close="$emit('close')"
  >
    <div class="space-y-4">
      <div v-if="serviceId === 'new'">
        <label class="block text-sm font-medium mb-1"> Group </label>
        <select
          v-model="selectedGroupId"
          class="v-select"
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
        <label class="block text-sm font-medium mb-1"> Project Type </label>
        <select
          v-model="form.type"
          class="v-select"
        >
          <option value="dotnet">.NET Run</option>
          <option value="npm">NPM Run Dev</option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium mb-1"> Name </label>
        <input
          v-model="form.name"
          type="text"
          class="v-input"
        />
      </div>

      <div>
        <label class="block text-sm font-medium mb-1"> Path </label>
        <input
          v-model="form.path"
          type="text"
          class="v-input"
        />
      </div>

      <EnvVariables
        v-model="form.env"
        :inherited-env="inheritedEnv"
      />
    </div>

    <template #footer>
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
    </template>
  </VDialog>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { map } from "lodash-es";
import { useServicesStore } from "@/stores/services";
import EnvVariables from "./EnvVariables.vue";
import VDialog from "./VDialog.vue";

const store = useServicesStore();

const props = defineProps<{
  serviceId: string;
}>();

const emit = defineEmits<{
  close: [];
}>();
const form = ref(setup());
const selectedGroupId = ref("");

const inheritedEnv = computed(() => {
  if (props.serviceId === "new") {
    if (!selectedGroupId.value) return {};
    return store.groups[selectedGroupId.value]?.env || {};
  }

  const service = store.services[props.serviceId];
  return service?.inheritedEnv || {};
});

function setup() {
  const value = props.serviceId;
  if (value === "new") {
    return { name: "", path: "", env: [], type: "dotnet" };
  }
  const service = store.services[value];
  if (!service) {
    throw new Error(`Service with id ${value} not found`);
  }
  return {
    name: service.name,
    path: service.path,
    type: service.type || "dotnet",
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
    type: form.value.type,
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
