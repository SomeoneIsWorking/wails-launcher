import { ref } from 'vue';
import { defineStore } from 'pinia';

interface ConfirmOptions {
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  confirmVariant?: 'danger' | 'primary';
}

export const useConfirmStore = defineStore('confirm', () => {
  const visible = ref(false);
  const title = ref('Confirm');
  const message = ref('');
  const confirmText = ref('Confirm');
  const cancelText = ref('Cancel');
  const confirmVariant = ref<'danger' | 'primary'>('primary');
  
  let resolvePromise: ((value: boolean) => void) | null = null;

  function show(options: ConfirmOptions): Promise<boolean> {
    title.value = options.title ?? 'Confirm';
    message.value = options.message;
    confirmText.value = options.confirmText ?? 'Confirm';
    cancelText.value = options.cancelText ?? 'Cancel';
    confirmVariant.value = options.confirmVariant ?? 'primary';
    visible.value = true;

    return new Promise<boolean>((resolve) => {
      resolvePromise = resolve;
    });
  }

  function confirm() {
    visible.value = false;
    if (resolvePromise) {
      resolvePromise(true);
      resolvePromise = null;
    }
  }

  function cancel() {
    visible.value = false;
    if (resolvePromise) {
      resolvePromise(false);
      resolvePromise = null;
    }
  }

  return {
    visible,
    title,
    message,
    confirmText,
    cancelText,
    confirmVariant,
    show,
    confirm,
    cancel,
  };
});

// Utility function for easy access
export async function confirmDialog(options: ConfirmOptions): Promise<boolean> {
  const store = useConfirmStore();
  return store.show(options);
}
