<template>
  <div>
    <label class="block text-sm font-medium mb-2">
      Environment Variables
    </label>

    <div v-if="inheritedEnv && Object.keys(inheritedEnv).length > 0" class="mb-4">
      <div class="text-[10px] font-bold text-gray-400 uppercase tracking-widest mb-1.5 flex items-center gap-1 ml-1">
        Inherited from Group
      </div>
      <div class="bg-gray-50 border border-gray-100 rounded-lg p-2.5 space-y-2 overflow-y-auto">
        <div v-for="(value, key) in inheritedEnv" :key="key" 
             class="flex flex-col group transition-opacity"
             :class="{'opacity-50': isProjectOverwriting(key)}">
          <span class="font-mono font-bold text-gray-500 text-[10px] break-all" 
                :class="{'line-through decoration-gray-400 decoration-1': isProjectOverwriting(key)}"
                :title="isProjectOverwriting(key) ? `Overwritten by project variable` : String(key)">
            {{ key }}
          </span>
          <span class="font-mono text-gray-600 text-[11px] break-all pl-2 border-l border-gray-200"
                :class="{'line-through decoration-gray-300': isProjectOverwriting(key)}">
            {{ value }}
          </span>
        </div>
      </div>
    </div>

    <div v-if="inheritedEnv && Object.keys(inheritedEnv).length > 0" class="text-[10px] font-bold text-gray-400 uppercase tracking-widest mb-2 flex items-center gap-1 ml-1">
      Project Variables
    </div>

    <div class="space-y-2">
      <div v-for="item in modelValue" :key="item.index" class="flex flex-col gap-1">
        <div class="flex gap-2">
          <div class="flex-1 relative">
            <input
              v-model="item.key"
              placeholder="Key"
              class="v-input w-full"
              :class="{'!border-indigo-200 !bg-indigo-50/30': isInherited(item.key)}"
            />
            <span v-if="item.key && isInherited(item.key)" class="absolute right-2 top-1.5 text-[8px] font-bold text-indigo-400 uppercase pointer-events-none">Override</span>
          </div>
          <div class="flex-1 relative">
            <input
              v-model="item.value"
              placeholder="Value"
              class="v-input w-full"
            />
            <span v-if="item.key && item.value === '' && isInherited(item.key)" 
                  class="absolute right-2 top-1.5 text-[8px] font-bold text-red-400 uppercase pointer-events-none bg-white px-1">
              Unset
            </span>
          </div>
          <button
            @click="removeEnvVar(item.index)"
            class="px-2 py-1 text-red-500 hover:bg-red-50 rounded flex-shrink-0"
          >
            <XIcon :size="16" />
          </button>
        </div>
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
  inheritedEnv?: Record<string, string>;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  "update:modelValue": [value: EnvVar[]];
}>();

const isInherited = (key: string) => {
  if (!key || !props.inheritedEnv) return false;
  return key in props.inheritedEnv;
};

const isProjectOverwriting = (key: string) => {
  if (!key) return false;
  return props.modelValue.some((item) => item.key === key);
};

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