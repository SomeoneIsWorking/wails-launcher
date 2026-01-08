<template>
  <VDialog
    title="Import Project"
    @close="$emit('close')"
  >
    <template #title>
      <DownloadIcon :size="20" class="text-indigo-600" />
      Import Project
    </template>

    <div class="space-y-6">
      <!-- Tabs -->
      <div
        class="flex border-b border-gray-100 bg-gray-50/50 p-1 rounded-xl flex-shrink-0"
      >
        <button
          v-for="tab in [
            { id: 'sln', label: 'Solution' },
            { id: 'npm', label: 'package.json' },
            { id: 'dotnet', label: '.csproj' },
          ]"
          :key="tab.id"
          @click="importTab = tab.id as any"
          :class="[
            'flex-1 py-2 text-sm font-semibold rounded-lg transition-all',
            importTab === tab.id
              ? 'bg-white text-indigo-600 shadow-sm ring-1 ring-black/5'
              : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100/50',
          ]"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="space-y-4">
        <!-- Group Selection (for single files) -->
        <div v-if="importTab !== 'sln'">
          <label
            class="block text-xs font-bold text-gray-400 uppercase tracking-widest mb-2 ml-1"
          >
            Target Group
          </label>
          <select
            v-model="importGroupId"
            class="v-select"
          >
            <option value="" disabled>Select a group...</option>
            <option v-for="(group, id) in groups" :key="id" :value="id">
              {{ group.name }}
            </option>
          </select>
        </div>

        <!-- File Path -->
        <div>
          <label
            class="block text-xs font-bold text-gray-400 uppercase tracking-widest mb-2 ml-1"
          >
            {{
              importTab === "sln"
                ? "Solution"
                : importTab === "npm"
                ? "Package.json"
                : "Project"
            }}
            File Path
          </label>
          <div class="flex gap-2">
            <div class="relative flex-1">
              <input
                v-model="importPath"
                type="text"
                class="v-input"
                :placeholder="
                  importTab === 'sln'
                    ? '/path/to/solution.sln'
                    : importTab === 'npm'
                    ? 'package.json'
                    : 'project.csproj'
                "
              />
            </div>
            <button
              @click="browseFile"
              class="px-5 py-2.5 bg-white border border-gray-200 text-gray-700 rounded-xl hover:bg-gray-50 hover:border-gray-300 transition-all flex items-center gap-2 shadow-sm whitespace-nowrap active:scale-95"
            >
              <FolderOpenIcon :size="18" />
              <span class="text-sm font-semibold">Browse</span>
            </button>
          </div>
        </div>

        <p
          class="text-xs text-gray-400 leading-relaxed bg-blue-50/50 p-3 rounded-xl border border-blue-100/50"
        >
          <template v-if="importTab === 'sln'">
            Importing a solution will create a new group and automatically add
            all compatible .NET projects within it.
          </template>
          <template v-else-if="importTab === 'npm'">
            Importing a package.json will add it to the selected group as an NPM
            Dev project.
          </template>
          <template v-else>
            Importing a .csproj will add it to the selected group as a .NET Run
            project.
          </template>
        </p>
      </div>
    </div>

    <template #footer>
      <button
        @click="$emit('close')"
        class="flex-1 px-4 py-2.5 text-sm font-bold text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-xl transition-all"
      >
        Cancel
      </button>
      <button
        @click="handleImport"
        :disabled="!importPath || (importTab !== 'sln' && !importGroupId)"
        class="flex-[2] px-4 py-2.5 bg-indigo-600 text-white text-sm font-bold rounded-xl hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all shadow-lg shadow-indigo-200 active:scale-[0.98]"
      >
        Import Project
      </button>
    </template>
  </VDialog>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { storeToRefs } from "pinia";
import { useServicesStore } from "@/stores/services";
import { DownloadIcon, FolderOpenIcon } from "lucide-vue-next";
import VDialog from "./VDialog.vue";

const emit = defineEmits<{
  close: [];
}>();

const store = useServicesStore();
const { groups } = storeToRefs(store);

const importTab = ref<"sln" | "npm" | "dotnet">("sln");
const importPath = ref("");
const importGroupId = ref("");

async function browseFile() {
  let title = "Select File";
  let filterName = "All Files";
  let pattern = "*.*";

  if (importTab.value === "sln") {
    title = "Select Solution File";
    filterName = "Solution Files (*.sln)";
    pattern = "*.sln";
  } else if (importTab.value === "npm") {
    title = "Select package.json";
    filterName = "package.json";
    pattern = "*.json"; // Multiple OS's prefer extension-based filters
  } else if (importTab.value === "dotnet") {
    title = "Select Project File";
    filterName = "Project Files (*.csproj)";
    pattern = "*.csproj";
  }

  const path = await store.browse(title, filterName, pattern);
  if (path) {
    importPath.value = path;
  }
}

async function handleImport() {
  if (!importPath.value) return;

  try {
    if (importTab.value === "sln") {
      await store.importSLN(importPath.value);
    } else {
      if (!importGroupId.value) return;
      await store.importProject(
        importGroupId.value,
        importPath.value,
        importTab.value
      );
    }
    emit("close");
  } catch (error) {
    console.error("Failed to import:", error);
  }
}
</script>
