import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { deviceHeartbeatMutation } from '@/lib/api/@tanstack/react-query.gen';
import {
  getAutoHeartbeatSettings,
  storeClientIp,
  SETTINGS_EVENT,
  type AutoHeartbeatSettings,
} from '@/lib/autoHeartbeat';
import { useAuth } from '@/features/auth/hooks/useAuth';

export function useAutoHeartbeat() {
  const { isAuthenticated } = useAuth();
  const [settings, setSettings] = useState<AutoHeartbeatSettings | null>(
    getAutoHeartbeatSettings,
  );
  const [lastIp, setLastIp] = useState<string | null>(null);

  // Sync settings from localStorage (same-tab via SETTINGS_EVENT, cross-tab via native storage)
  useEffect(() => {
    const handler = () => setSettings(getAutoHeartbeatSettings());
    window.addEventListener(SETTINGS_EVENT, handler);
    window.addEventListener('storage', handler);
    return () => {
      window.removeEventListener(SETTINGS_EVENT, handler);
      window.removeEventListener('storage', handler);
    };
  }, []);

  const { mutate } = useMutation({
    ...deviceHeartbeatMutation(),
    onSuccess: (address) => {
      setLastIp(address.ip);
      storeClientIp(address.ip);
    },
    // silent — no toast
  });

  const active = isAuthenticated && settings;
  const deviceId = settings?.deviceId;
  const intervalSeconds = settings?.intervalSeconds;

  // Main interval
  useEffect(() => {
    if (!active || !deviceId || !intervalSeconds) return;
    const fire = () => mutate({ path: { device_id: deviceId } });
    fire();
    const id = setInterval(fire, intervalSeconds * 1_000);
    return () => clearInterval(id);
  }, [active, mutate, deviceId, intervalSeconds]);

  // Re-fire on tab focus
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible' && active && deviceId)
        mutate({ path: { device_id: deviceId } });
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [active, mutate, deviceId]);

  // Only expose IP when heartbeat is active; stale IP is hidden, not leaked
  const clientIp = active ? lastIp : null;

  return { clientIp, activeDeviceId: deviceId ?? null };
}
