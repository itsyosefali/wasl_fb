<script setup lang="ts">
import { computed } from 'vue';
import type { Page } from '@/api/gateway';

const props = withDefaults(
  defineProps<{
    pages?: Page[] | null;
    modelValue: string | null;
  }>(),
  { pages: () => [] },
);

const pages = computed(() => props.pages ?? []);

const emit = defineEmits<{
  'update:modelValue': [pageId: string];
}>();

const selectedPage = computed({
  get: () => props.modelValue || '',
  set: (v: string) => emit('update:modelValue', v),
});

const currentPage = computed(() =>
  pages.value.find((p) => p.id === props.modelValue),
);
</script>

<template>
  <div class="border-b border-n-weak px-4 py-4">
    <label v-if="pages.length > 1" class="mb-2 block text-xs font-medium text-n-slate-11">
      Inbox
    </label>
    <select
      v-if="pages.length > 1"
      v-model="selectedPage"
      class="mb-2 w-full rounded-lg border border-n-weak bg-n-solid-1 px-3 py-2 text-sm font-medium text-n-slate-12 outline-none focus:border-woot-500"
    >
      <option v-for="page in pages" :key="page.id" :value="page.id">
        {{ page.name }}
      </option>
    </select>
    <h1 class="text-base font-semibold text-n-slate-12">
      {{ currentPage?.name || 'Conversations' }}
    </h1>
    <p class="mt-0.5 text-xs text-n-slate-11">
      Facebook Messenger
      <span v-if="currentPage"> · Page ID {{ currentPage.meta_page_id }}</span>
    </p>
  </div>
</template>
