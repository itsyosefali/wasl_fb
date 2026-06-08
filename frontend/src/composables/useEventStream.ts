import { onMounted, onUnmounted, ref } from 'vue';
import { API_KEY, API_URL } from '@/api/gateway';

export function useEventStream(onEvent: () => void) {
  const connected = ref(false);
  let ws: WebSocket | null = null;

  onMounted(() => {
    const wsUrl =
      API_URL.replace(/^http/, 'ws') +
      '/events/stream?api_key=' +
      encodeURIComponent(API_KEY);
    ws = new WebSocket(wsUrl);
    ws.onopen = () => {
      connected.value = true;
    };
    ws.onclose = () => {
      connected.value = false;
    };
    ws.onmessage = () => onEvent();
  });

  onUnmounted(() => {
    ws?.close();
  });

  return { connected };
}
