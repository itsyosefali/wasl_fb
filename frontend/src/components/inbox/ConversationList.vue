<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import type { Conversation, Page } from '@/api/gateway';
import { formatTime } from '@/api/gateway';
import InboxSelector from '@/components/inbox/InboxSelector.vue';

const props = withDefaults(
  defineProps<{
    pages?: Page[] | null;
    selectedPageId: string | null;
    conversations?: Conversation[] | null;
    selectedKey: string | null;
    loading?: boolean;
  }>(),
  {
    pages: () => [],
    conversations: () => [],
  },
);

const pages = computed(() => props.pages ?? []);
const conversations = computed(() => props.conversations ?? []);

const emit = defineEmits<{
  select: [conversation: Conversation];
  'update:selectedPageId': [pageId: string];
}>();

function displayName(c: Conversation) {
  return c.contact_name || c.external_id || 'Unknown';
}
</script>

<template>
  <div class="flex h-full w-[360px] shrink-0 flex-col border-r border-n-weak bg-n-solid-1">
    <InboxSelector
      :pages="pages"
      :model-value="selectedPageId"
      @update:model-value="emit('update:selectedPageId', $event)"
    />

    <div v-if="pages.length === 0 && !loading" class="flex flex-1 flex-col items-center justify-center px-6 text-center text-sm text-n-slate-11">
      <p>No Facebook Page connected.</p>
      <RouterLink to="/connect" class="mt-3 text-woot-500 hover:underline">
        Connect a Facebook Page
      </RouterLink>
    </div>

    <div v-else-if="loading" class="flex flex-1 items-center justify-center text-sm text-n-slate-11">
      Loading…
    </div>

    <div
      v-else-if="conversations.length === 0"
      class="flex flex-1 flex-col items-center justify-center px-6 text-center text-sm text-n-slate-11"
    >
      <p>No conversations yet for this page.</p>
      <p class="mt-2 text-xs">
        Send a message to your Page on Messenger, or set up webhooks (ngrok) so Meta can deliver events.
      </p>
    </div>

    <div v-else class="flex-1 overflow-y-auto">
      <button
        v-for="conv in conversations"
        :key="`${conv.contact_id}:${conv.page_id}`"
        type="button"
        class="cw-conversation-card"
        :class="{ active: selectedKey === `${conv.contact_id}:${conv.page_id}` }"
        @click="emit('select', conv)"
      >
        <div
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-woot-50 text-sm font-semibold text-woot-600"
        >
          {{ displayName(conv).charAt(0).toUpperCase() }}
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-start justify-between gap-2">
            <span class="truncate text-sm font-medium text-n-slate-12">
              {{ displayName(conv) }}
            </span>
            <span class="shrink-0 text-[11px] text-n-slate-11">
              {{ formatTime(conv.last_message_at) }}
            </span>
          </div>
          <div class="truncate text-xs text-n-slate-11">{{ conv.page_name }}</div>
          <div class="mt-1 truncate text-sm text-n-slate-11">
            {{ conv.last_message || 'No messages' }}
          </div>
        </div>
      </button>
    </div>
  </div>
</template>
