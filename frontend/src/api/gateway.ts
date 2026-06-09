function resolveApiUrl(): string {
  const configured = import.meta.env.VITE_API_URL?.trim();
  if (configured) return configured;
  if (typeof window !== 'undefined') return window.location.origin;
  return 'http://localhost:8080';
}

export const API_URL = resolveApiUrl();
export const API_KEY = import.meta.env.VITE_API_KEY || 'demo-api-key-change-in-production';
export const META_APP_ID = import.meta.env.VITE_META_APP_ID || '';

export const FB_SCOPES = [
  'pages_manage_metadata',
  'business_management',
  'pages_messaging',
  'pages_show_list',
  'pages_read_engagement',
].join(',');

export interface Page {
  id: string;
  meta_page_id: string;
  name: string;
  status: string;
  created_at?: string;
}

export interface PageDetail {
  id: string;
  name: string;
  access_token: string;
  exists: boolean;
}

export interface Conversation {
  contact_id: string;
  page_id: string;
  contact_name: string;
  external_id: string;
  platform: string;
  page_name: string;
  meta_page_id: string;
  last_message: string;
  last_direction: string;
  last_message_at: string;
}

export interface ThreadMessage {
  id: string;
  external_id?: string;
  direction: 'in' | 'out';
  message: string;
  created_at: string;
}

async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
      ...(init.headers || {}),
    },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || res.statusText);
  }
  return res.json() as Promise<T>;
}

export function listPages() {
  return api<{ data: Page[] }>('/pages');
}

export function listFacebookPages(userAccessToken: string) {
  return api<{ user_access_token: string; page_details: PageDetail[] }>(
    '/auth/facebook/pages',
    {
      method: 'POST',
      body: JSON.stringify({ user_access_token: userAccessToken }),
    },
  );
}

export function registerFacebookPage(payload: {
  user_access_token: string;
  page_access_token: string;
  page_id: string;
  name: string;
}) {
  return api<Page>('/auth/facebook/register', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function listConversations(pageId?: string) {
  const qs = pageId ? `?page_id=${encodeURIComponent(pageId)}` : '';
  return api<{ data: Conversation[] }>(`/conversations${qs}`);
}

export function listThreadMessages(contactId: string, pageId: string) {
  return api<{ data: ThreadMessage[] }>(
    `/conversations/${contactId}/messages?page_id=${pageId}`,
  );
}

export function sendMessage(pageId: string, recipientId: string, text: string) {
  return api('/messages/send', {
    method: 'POST',
    body: JSON.stringify({
      channel: 'facebook',
      page_id: pageId,
      recipient_id: recipientId,
      text,
    }),
  });
}

export function oauthRedirectUrl() {
  return `${API_URL}/auth/facebook?api_key=${encodeURIComponent(API_KEY)}`;
}

export function formatTime(iso: string) {
  if (!iso) return '';
  const d = new Date(iso);
  const now = new Date();
  const sameDay =
    d.getDate() === now.getDate() &&
    d.getMonth() === now.getMonth() &&
    d.getFullYear() === now.getFullYear();
  if (sameDay) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

export function conversationKey(c: Conversation) {
  return `${c.contact_id}:${c.page_id}`;
}
