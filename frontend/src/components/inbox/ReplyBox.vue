<script setup lang="ts">
import { ref } from 'vue';

const props = defineProps<{
  disabled?: boolean;
}>();

const emit = defineEmits<{
  send: [text: string];
}>();

const text = ref('');

function submit() {
  const value = text.value.trim();
  if (!value || props.disabled) return;
  emit('send', value);
  text.value = '';
}
</script>

<template>
  <div class="cw-reply-box">
    <div class="border-b border-n-weak px-4 py-2">
      <span class="text-xs font-medium text-n-slate-11">Reply</span>
    </div>
    <div class="px-4 py-3">
      <textarea
        v-model="text"
        rows="3"
        class="w-full resize-none border-0 bg-transparent text-sm text-n-slate-12 outline-none placeholder:text-n-slate-11"
        placeholder="Shift + enter for new line. Enter to send"
        :disabled="disabled"
        @keydown.enter.exact.prevent="submit"
      />
    </div>
    <div class="flex items-center justify-end gap-2 border-t border-n-weak px-4 py-3">
      <button type="button" class="cw-btn-primary" :disabled="disabled || !text.trim()" @click="submit">
        Send
      </button>
    </div>
  </div>
</template>
