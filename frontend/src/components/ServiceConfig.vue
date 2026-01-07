<template>
  <div class="bg-white p-6 rounded-lg shadow-lg w-auto min-w-[30rem]">
    <h3 class="text-lg font-medium mb-4">
      {{ serviceId === "new" ? "Add" : "Edit" }} Service
    </h3>

    <div class="space-y-4">
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

      <div>
        <label class="block text-sm font-medium mb-2">
          Environment Variables
        </label>
        <div class="space-y-2">
          <div v-for="item in form.env" :key="item.index" class="flex gap-2">
            <input
              v-model="item.key"
              placeholder="Key"
              class="flex-1 px-3 py-2 border rounded"
            />
            <input
              v-model="item.value"
              placeholder="Value"
              class="flex-1 px-3 py-2 border rounded"
            />
            <button
              @click="
                form.env = form.env.filter(({ index }) => index !== item.index)
              "
              class="px-2 py-1 text-red-500 hover:bg-red-50 rounded"
            >
              <XIcon :size="16" />
            </button>
          </div>
        </div>
        <button
          @click="
            form.env.push({
              index: (max(form.env.map((x) => x.index)) ?? 0) + 1,
              key: '',
              value: '',
            })
          "
          class="mt-2 text-sm text-blue-500 hover:text-blue-600"
        >
          + Add Environment Variable
        </button>
      </div>
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
import { XIcon } from "lucide-vue-next";
import { map, max } from "lodash-es";
import { useServicesStore } from "@/stores/services";

const store = useServicesStore();

const props = defineProps<{
  serviceId: string;
}>();

const emit = defineEmits<{
  close: [];
}>();
const form = ref(setup());

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
      await store.addService(toModel());
    } else {
      await store.updateService(props.serviceId, toModel());
    }
    emit("close");
  } catch (error) {
    console.error("Failed to save service:", error);
  }
}
</script>
