<script setup lang="ts">
import type { Conversation } from '@/api/gateway';
import MessageList from '@/components/inbox/MessageList.vue';
import ReplyBox from '@/components/inbox/ReplyBox.vue';

defineProps<{
  conversation: Conversation | null;
  messages: import('@/api/gateway').ThreadMessage[];
  loading?: boolean;
  sending?: boolean;
  streamConnected?: boolean;
}>();

const emit = defineEmits<{
  send: [text: string];
}>();

function displayName(c: Conversation) {
  return c.contact_name || c.external_id || 'Unknown';
}
</script>

<template>
  <div class="flex h-full min-w-0 flex-1 flex-col bg-n-solid-2">
    <template v-if="!conversation">
      <div class="flex flex-1 flex-col items-center justify-center text-n-slate-11">
        <svg
          class="mb-4 opacity-40"
          xmlns="http://www.w3.org/2000/svg"
          width="48"
          height="48"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
        >
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
        <p class="text-sm">Select a conversation from the list</p>
      </div>
    </template>

    <template v-else>
      <header class="flex items-center justify-between border-b border-n-weak bg-n-solid-1 px-6 py-4">
        <div>
          <h2 class="text-base font-semibold text-n-slate-12">
            {{ displayName(conversation) }}
          </h2>
          <p class="text-xs text-n-slate-11">
            via {{ conversation.page_name }}
            <span class="mx-1">·</span>
            PSID {{ conversation.external_id }}
          </p>
        </div>
        <div class="flex items-center gap-2 text-xs text-n-slate-11">
          <span
            class="inline-block h-2 w-2 rounded-full"
            :class="streamConnected ? 'bg-green-500' : 'bg-n-strong'"
          />
          {{ streamConnected ? 'Live' : 'Offline' }}
        </div>
      </header>

      <MessageList :messages="messages" :loading="loading" />
      <ReplyBox :disabled="sending" @send="emit('send', $event)" />
    </template>
  </div>
</template>
