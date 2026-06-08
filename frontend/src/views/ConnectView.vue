<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import {
  META_APP_ID,
  oauthRedirectUrl,
  listFacebookPages,
  registerFacebookPage,
  type PageDetail,
} from '@/api/gateway';
import { facebookLogin, initFacebookSDK } from '@/composables/useFacebookSDK';

const router = useRouter();

const userAccessToken = ref('');
const pageList = ref<PageDetail[]>([]);
const selectedPageId = ref('');
const pageName = ref('');
const isCreating = ref(false);
const hasError = ref(false);
const errorMessage = ref('');
const hasLoginStarted = ref(false);
const loadingPages = ref(false);

const selectablePages = ref<PageDetail[]>([]);

function syncSelectable() {
  selectablePages.value = pageList.value.filter((p) => !p.exists);
}

async function fetchPages(token: string) {
  loadingPages.value = true;
  hasError.value = false;
  try {
    const data = await listFacebookPages(token);
    userAccessToken.value = data.user_access_token;
    pageList.value = data.page_details;
    syncSelectable();
  } catch (e) {
    hasError.value = true;
    errorMessage.value = e instanceof Error ? e.message : 'Failed to load pages';
  } finally {
    loadingPages.value = false;
  }
}

async function startLogin() {
  hasLoginStarted.value = true;
  hasError.value = false;
  try {
    await initFacebookSDK();
    const token = await facebookLogin();
    await fetchPages(token);
  } catch (e) {
    hasError.value = true;
    errorMessage.value = e instanceof Error ? e.message : 'Facebook login failed';
  }
}

function onPageSelect(pageId: string) {
  selectedPageId.value = pageId;
  const page = pageList.value.find((p) => p.id === pageId);
  pageName.value = page?.name || '';
}

async function createChannel() {
  const page = pageList.value.find((p) => p.id === selectedPageId.value);
  if (!page || !userAccessToken.value) return;

  isCreating.value = true;
  hasError.value = false;
  try {
    await registerFacebookPage({
      user_access_token: userAccessToken.value,
      page_access_token: page.access_token,
      page_id: page.id,
      name: pageName.value || page.name,
    });
    page.exists = true;
    syncSelectable();
    await router.push('/');
  } catch (e) {
    hasError.value = true;
    errorMessage.value = e instanceof Error ? e.message : 'Failed to connect page';
  } finally {
    isCreating.value = false;
  }
}

onMounted(() => {
  if (META_APP_ID) {
    initFacebookSDK().catch(() => {});
  }
});
</script>

<template>
  <div class="h-full overflow-y-auto bg-n-solid-2">
    <div class="mx-auto max-w-2xl px-6 py-8">
      <header class="mb-6">
        <h1 class="text-xl font-semibold text-n-slate-12">Connect Facebook Page</h1>
        <p class="mt-1 text-sm text-n-slate-11">
          Same flow as Chatwoot: log in with Facebook, choose a page, start receiving Messenger
          conversations in your inbox.
        </p>
      </header>

      <div v-if="!hasLoginStarted && !userAccessToken" class="rounded-xl border border-n-weak bg-n-solid-1 p-6 shadow-card">
        <p class="mb-4 text-sm text-n-slate-11">
          Click below to authorize Meta Gateway to access your Facebook Pages and Messenger
          conversations.
        </p>
        <div class="flex flex-wrap gap-3">
          <button v-if="META_APP_ID" type="button" class="cw-btn-primary" @click="startLogin">
            Continue with Facebook
          </button>
          <a v-else class="cw-btn-secondary" :href="oauthRedirectUrl()">Server OAuth fallback</a>
          <a class="cw-btn-secondary" :href="oauthRedirectUrl()">OAuth redirect</a>
        </div>
        <p v-if="!META_APP_ID" class="mt-3 text-xs text-amber-700">
          Set <code>VITE_META_APP_ID</code> for the Facebook SDK button.
        </p>
      </div>

      <div v-if="loadingPages || isCreating" class="mt-6 rounded-xl border border-n-weak bg-n-solid-1 p-8 text-center text-sm text-n-slate-11">
        {{ isCreating ? 'Creating channel…' : 'Loading pages…' }}
      </div>

      <div v-if="hasError" class="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
        {{ errorMessage }}
      </div>

      <div
        v-if="userAccessToken && !loadingPages && selectablePages.length > 0"
        class="mt-6 rounded-xl border border-n-weak bg-n-solid-1 p-6 shadow-card"
      >
        <h2 class="mb-4 text-sm font-semibold text-n-slate-12">Choose a page</h2>

        <label class="mb-2 block text-xs font-medium text-n-slate-11">Facebook Page</label>
        <select
          v-model="selectedPageId"
          class="mb-4 w-full rounded-lg border border-n-weak bg-n-solid-1 px-3 py-2 text-sm outline-none focus:border-woot-500"
          @change="onPageSelect(selectedPageId)"
        >
          <option value="" disabled>Select a page</option>
          <option v-for="page in selectablePages" :key="page.id" :value="page.id">
            {{ page.name }}
          </option>
        </select>

        <label class="mb-2 block text-xs font-medium text-n-slate-11">Inbox name</label>
        <input
          v-model="pageName"
          type="text"
          class="mb-4 w-full rounded-lg border border-n-weak bg-n-solid-1 px-3 py-2 text-sm outline-none focus:border-woot-500"
          placeholder="Page inbox name"
        />

        <button
          type="button"
          class="cw-btn-primary"
          :disabled="!selectedPageId || isCreating"
          @click="createChannel"
        >
          Create channel
        </button>
      </div>

      <div
        v-if="userAccessToken && !loadingPages && selectablePages.length === 0 && pageList.length > 0"
        class="mt-6 rounded-xl border border-green-200 bg-green-50 p-6 text-sm text-green-800"
      >
        All available pages are already connected.
        <RouterLink to="/" class="ml-1 font-medium text-woot-600 hover:underline">Go to inbox</RouterLink>
      </div>
    </div>
  </div>
</template>
