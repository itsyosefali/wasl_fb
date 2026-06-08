<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import {
  conversationKey,
  listConversations,
  listPages,
  listThreadMessages,
  sendMessage,
  type Conversation,
  type Page,
  type ThreadMessage,
} from '@/api/gateway';
import { useEventStream } from '@/composables/useEventStream';
import ConversationList from '@/components/inbox/ConversationList.vue';
import ConversationPanel from '@/components/inbox/ConversationPanel.vue';

const STORAGE_KEY = 'meta-gateway-selected-page';

const pages = ref<Page[]>([]);
const selectedPageId = ref<string | null>(null);
const conversations = ref<Conversation[]>([]);
const selected = ref<Conversation | null>(null);
const selectedKey = ref<string | null>(null);
const messages = ref<ThreadMessage[]>([]);
const loadingList = ref(true);
const loadingThread = ref(false);
const sending = ref(false);

function pickDefaultPage(list: Page[]): string | null {
  if (list.length === 0) return null;
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved && list.some((p) => p.id === saved)) return saved;
  const sorted = [...list].sort((a, b) => {
    const ta = a.created_at ? Date.parse(a.created_at) : 0;
    const tb = b.created_at ? Date.parse(b.created_at) : 0;
    return tb - ta;
  });
  // Prefer real pages over the old verify-script demo page id
  const real = sorted.find((p) => p.meta_page_id !== '123456789');
  return (real || sorted[0]).id;
}

async function loadPages() {
  const res = await listPages();
  pages.value = res.data;
  if (!selectedPageId.value) {
    selectedPageId.value = pickDefaultPage(pages.value);
  }
}

async function refreshConversations() {
  if (!selectedPageId.value) {
    conversations.value = [];
    loadingList.value = false;
    return;
  }
  try {
    const res = await listConversations(selectedPageId.value);
    conversations.value = res.data ?? [];
    if (selected.value && selected.value.page_id !== selectedPageId.value) {
      selected.value = null;
      selectedKey.value = null;
      messages.value = [];
    }
  } finally {
    loadingList.value = false;
  }
}

async function refreshThread() {
  if (!selected.value) return;
  loadingThread.value = true;
  try {
    const res = await listThreadMessages(
      selected.value.contact_id,
      selected.value.page_id,
    );
    messages.value = res.data;
  } finally {
    loadingThread.value = false;
  }
}

function selectConversation(conv: Conversation) {
  selected.value = conv;
  selectedKey.value = conversationKey(conv);
}

async function onSend(text: string) {
  if (!selected.value) return;
  sending.value = true;
  try {
    await sendMessage(selected.value.page_id, selected.value.external_id, text);
    await refreshThread();
    await refreshConversations();
  } finally {
    sending.value = false;
  }
}

const { connected: streamConnected } = useEventStream(() => {
  void refreshConversations();
  void refreshThread();
});

watch(selectedPageId, (id) => {
  if (id) localStorage.setItem(STORAGE_KEY, id);
  selected.value = null;
  selectedKey.value = null;
  messages.value = [];
  loadingList.value = true;
  void refreshConversations();
});

watch(selected, () => {
  void refreshThread();
});

onMounted(async () => {
  await loadPages();
  await refreshConversations();
});
</script>

<template>
  <div class="flex h-full min-w-0">
    <ConversationList
      :pages="pages"
      :selected-page-id="selectedPageId"
      :conversations="conversations"
      :selected-key="selectedKey"
      :loading="loadingList"
      @select="selectConversation"
      @update:selected-page-id="selectedPageId = $event"
    />
    <ConversationPanel
      :conversation="selected"
      :messages="messages"
      :loading="loadingThread"
      :sending="sending"
      :stream-connected="streamConnected"
      @send="onSend"
    />
  </div>
</template>
