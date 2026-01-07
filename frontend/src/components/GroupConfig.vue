<template>
  <div class="bg-white p-6 rounded-lg shadow-lg w-auto min-w-[30rem]">
    <h3 class="text-lg font-medium mb-4">
      {{ groupId === "new" ? "Add" : "Edit" }} Group
    </h3>

    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium mb-1">Name</label>
        <input
          v-model="formData.name"
          type="text"
          class="w-full px-3 py-2 border rounded"
        />
      </div>

      <EnvVariables v-model="formData.env" />
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
import { ref, onMounted, watch } from "vue";
import { useServicesStore } from "@/stores/services";
import EnvVariables, { type EnvVar } from "./EnvVariables.vue";

interface Props {
  groupId: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  close: [];
}>();

const store = useServicesStore();

interface FormData {
  name: string;
  env: EnvVar[];
}

const formData = ref<FormData>({
  name: "",
  env: [],
});

function loadGroup() {
  if (props.groupId === "new") {
    formData.value = {
      name: "",
      env: [],
    };
  } else {
    const group = store.groups[props.groupId];
    if (group) {
      formData.value = {
        name: group.name,
        env: Object.entries(group.env).map(([key, value], index) => ({
          index,
          key,
          value: value as string,
        })),
      };
    }
  }
}

async function save() {
  const env = Object.fromEntries(
    formData.value.env
      .filter((e) => e.key.trim())
      .map((e) => [e.key, e.value])
  );

  try {
    if (props.groupId === "new") {
      await store.addGroup(formData.value.name, env);
    } else {
      await store.updateGroup(props.groupId, formData.value.name, env);
    }
    emit("close");
  } catch (error) {
    console.error("Failed to save group:", error);
  }
}

onMounted(() => {
  loadGroup();
});

watch(() => props.groupId, () => {
  loadGroup();
});
</script>