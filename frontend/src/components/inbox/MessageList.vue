<script setup lang="ts">
import type { ThreadMessage } from '@/api/gateway';
import { formatTime } from '@/api/gateway';
import { nextTick, ref, watch } from 'vue';

const props = defineProps<{
  messages: ThreadMessage[];
  loading?: boolean;
}>();

const panel = ref<HTMLElement | null>(null);

watch(
  () => props.messages.length,
  async () => {
    await nextTick();
    if (panel.value) {
      panel.value.scrollTop = panel.value.scrollHeight;
    }
  },
);
</script>

<template>
  <div ref="panel" class="flex-1 overflow-y-auto px-6 py-4">
    <div v-if="loading" class="py-8 text-center text-sm text-n-slate-11">
      Loading messages…
    </div>
    <div v-else-if="messages.length === 0" class="py-8 text-center text-sm text-n-slate-11">
      No messages in this conversation yet.
    </div>
    <div v-else class="flex flex-col gap-3">
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="flex flex-col"
        :class="msg.direction === 'out' ? 'items-end' : 'items-start'"
      >
        <div :class="msg.direction === 'out' ? 'cw-message-out' : 'cw-message-in'">
          {{ msg.message }}
        </div>
        <span class="mt-1 px-1 text-[11px] text-n-slate-11">
          {{ formatTime(msg.created_at) }}
        </span>
      </div>
    </div>
  </div>
</template>
