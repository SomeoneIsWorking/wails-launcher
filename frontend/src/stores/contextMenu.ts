import { ref } from 'vue';
import { defineStore } from 'pinia';

interface ContextMenuOption {
  label: string;
  action: () => void;
  disabled?: boolean;
}

export const useContextMenuStore = defineStore('contextMenu', () => {
  const visible = ref(false);
  const x = ref(0);
  const y = ref(0);
  const options = ref<ContextMenuOption[]>([]);

  function show(event: MouseEvent, menuOptions: ContextMenuOption[]) {
    if (menuOptions.length === 0) return;
    
    x.value = event.clientX;
    y.value = event.clientY;
    options.value = menuOptions;
    visible.value = true;
  }

  function hide() {
    visible.value = false;
  }

  return {
    visible,
    x,
    y,
    options,
    show,
    hide,
  };
});
