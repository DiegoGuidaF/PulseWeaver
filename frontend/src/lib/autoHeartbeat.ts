const SETTINGS_KEY = 'wallydic_auto_heartbeat';
const CLIENT_IP_KEY = 'wallydic_client_ip';

export const CLIENT_IP_EVENT = 'wallydic:client-ip-updated';

export interface AutoHeartbeatSettings {
  deviceId: number;
  intervalSeconds: number;
}

export function getAutoHeartbeatSettings(): AutoHeartbeatSettings | null {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY);
    return raw ? (JSON.parse(raw) as AutoHeartbeatSettings) : null;
  } catch {
    return null;
  }
}

export function setAutoHeartbeatSettings(s: AutoHeartbeatSettings): void {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
  window.dispatchEvent(new Event('storage'));
}

export function clearAutoHeartbeatSettings(): void {
  localStorage.removeItem(SETTINGS_KEY);
  window.dispatchEvent(new Event('storage'));
}

export function getStoredClientIp(): string | null {
  return localStorage.getItem(CLIENT_IP_KEY);
}

export function storeClientIp(ip: string): void {
  localStorage.setItem(CLIENT_IP_KEY, ip);
  window.dispatchEvent(new CustomEvent(CLIENT_IP_EVENT, { detail: ip }));
}
