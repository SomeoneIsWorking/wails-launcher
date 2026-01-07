<template>
  <div>
    <label class="block text-sm font-medium mb-2">
      Environment Variables
    </label>
    <div class="space-y-2">
      <div v-for="item in modelValue" :key="item.index" class="flex gap-2">
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
          @click="removeEnvVar(item.index)"
          class="px-2 py-1 text-red-500 hover:bg-red-50 rounded"
        >
          <XIcon :size="16" />
        </button>
      </div>
    </div>
    <button
      @click="addEnvVar"
      class="mt-2 text-sm text-blue-500 hover:text-blue-600"
    >
      + Add Environment Variable
    </button>
  </div>
</template>

<script setup lang="ts">
import { XIcon } from "lucide-vue-next";
import { max } from "lodash-es";

export interface EnvVar {
  index: number;
  key: string;
  value: string;
}

interface Props {
  modelValue: EnvVar[];
}

const props = defineProps<Props>();
const emit = defineEmits<{
  "update:modelValue": [value: EnvVar[]];
}>();

function addEnvVar() {
  const newIndex = (max(props.modelValue.map((x) => x.index)) ?? 0) + 1;
  const newEnvVars = [...props.modelValue, { index: newIndex, key: "", value: "" }];
  emit("update:modelValue", newEnvVars);
}

function removeEnvVar(index: number) {
  const newEnvVars = props.modelValue.filter((item) => item.index !== index);
  emit("update:modelValue", newEnvVars);
}
</script>