<script setup lang="ts">
import { useRoute } from 'vue-router';
import { RouterLink } from 'vue-router';

const route = useRoute();
</script>

<template>
  <aside
    class="flex w-16 shrink-0 flex-col items-center border-r border-n-weak bg-n-solid-1 py-4"
  >
    <div
      class="mb-6 flex h-10 w-10 items-center justify-center rounded-xl bg-woot-500 text-sm font-bold text-white"
      title="Meta Gateway"
    >
      MG
    </div>

    <nav class="flex flex-1 flex-col gap-2">
      <RouterLink
        to="/"
        class="cw-sidebar-item"
        :class="{ active: route.name === 'inbox' }"
        title="Inbox"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </RouterLink>
      <RouterLink
        to="/connect"
        class="cw-sidebar-item"
        :class="{ active: route.name === 'connect' }"
        title="Connect Facebook"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 2h-3a5 5 0 0 0-5 5v3H7v4h3v8h4v-8h3l1-4h-4V7a1 1 0 0 1 1-1h3z" />
        </svg>
      </RouterLink>
    </nav>
  </aside>
</template>
