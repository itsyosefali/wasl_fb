import { createRouter, createWebHistory } from 'vue-router';
import InboxView from '@/views/InboxView.vue';
import ConnectView from '@/views/ConnectView.vue';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'inbox', component: InboxView },
    { path: '/connect', name: 'connect', component: ConnectView },
  ],
});
