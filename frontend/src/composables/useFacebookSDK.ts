import { META_APP_ID, FB_SCOPES } from '@/api/gateway';

declare global {
  interface Window {
    FB?: {
      init: (opts: Record<string, unknown>) => void;
      login: (
        cb: (res: { status: string; authResponse?: { accessToken: string } }) => void,
        opts: Record<string, unknown>,
      ) => void;
    };
    fbAsyncInit?: () => void;
    fbSDKLoaded?: boolean;
  }
}

function loadScript(src: string, id: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.getElementById(id)) {
      resolve();
      return;
    }
    const script = document.createElement('script');
    script.id = id;
    script.src = src;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Failed to load Facebook SDK'));
    document.body.appendChild(script);
  });
}

export async function initFacebookSDK(): Promise<void> {
  if (!META_APP_ID) throw new Error('VITE_META_APP_ID is not set');
  if (window.fbSDKLoaded && window.FB) return;

  await loadScript('https://connect.facebook.net/en_US/sdk.js', 'facebook-jssdk');

  await new Promise<void>((resolve) => {
    window.fbAsyncInit = () => {
      window.FB?.init({
        appId: META_APP_ID,
        cookie: true,
        xfbml: false,
        version: 'v23.0',
        status: true,
      });
      window.fbSDKLoaded = true;
      resolve();
    };
    if (window.FB) {
      window.fbAsyncInit();
    }
  });
}

export function facebookLogin(): Promise<string> {
  return new Promise((resolve, reject) => {
    window.FB?.login(
      (response) => {
        if (response.status === 'connected' && response.authResponse?.accessToken) {
          resolve(response.authResponse.accessToken);
          return;
        }
        reject(new Error('Facebook login was cancelled or denied'));
      },
      { scope: FB_SCOPES },
    );
  });
}
