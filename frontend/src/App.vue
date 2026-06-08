<script setup lang="ts">
import { RouterView } from 'vue-router';
import AppSidebar from '@/components/layout/AppSidebar.vue';
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-n-solid-2">
    <AppSidebar />
    <main class="min-w-0 flex-1">
      <RouterView />
    </main>
  </div>
</template>
