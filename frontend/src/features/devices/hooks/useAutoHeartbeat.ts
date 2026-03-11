import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { deviceHeartbeatMutation } from '@/lib/api/@tanstack/react-query.gen';
import {
  getAutoHeartbeatSettings,
  storeClientIp,
  SETTINGS_EVENT,
  type AutoHeartbeatSettings,
} from '@/lib/autoHeartbeat';
import { useAuth } from '@/features/auth/AuthContext';

export function useAutoHeartbeat() {
  const { isAuthenticated } = useAuth();
  const [settings, setSettings] = useState<AutoHeartbeatSettings | null>(
    getAutoHeartbeatSettings,
  );
  const [clientIp, setClientIp] = useState<string | null>(null);

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
      setClientIp(address.ip);
      storeClientIp(address.ip);
    },
    // silent — no toast
  });

  // Main interval
  useEffect(() => {
    if (!isAuthenticated || !settings) {
      setClientIp(null);
      return;
    }
    const fire = () => mutate({ path: { device_id: settings.deviceId } });
    fire();
    const id = setInterval(fire, settings.intervalSeconds * 1_000);
    return () => clearInterval(id);
  }, [isAuthenticated, mutate, settings?.deviceId, settings?.intervalSeconds]);

  // Re-fire on tab focus
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible' && isAuthenticated && settings)
        mutate({ path: { device_id: settings.deviceId } });
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [isAuthenticated, mutate, settings?.deviceId]);

  return { clientIp, activeDeviceId: settings?.deviceId ?? null };
}
