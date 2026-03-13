import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getAutoHeartbeatSettings,
  setAutoHeartbeatSettings,
  clearAutoHeartbeatSettings,
  getStoredClientIp,
  storeClientIp,
  SETTINGS_EVENT,
  CLIENT_IP_EVENT,
} from './autoHeartbeat';

describe('autoHeartbeat', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('getAutoHeartbeatSettings', () => {
    it('returns null when key is absent', () => {
      expect(getAutoHeartbeatSettings()).toBeNull();
    });

    it('returns parsed settings when key exists', () => {
      localStorage.setItem(
        'pulseweaver_auto_heartbeat',
        JSON.stringify({ deviceId: 7, intervalSeconds: 60 }),
      );
      expect(getAutoHeartbeatSettings()).toEqual({ deviceId: 7, intervalSeconds: 60 });
    });

    it('returns null on invalid JSON', () => {
      localStorage.setItem('pulseweaver_auto_heartbeat', 'not-json{');
      expect(getAutoHeartbeatSettings()).toBeNull();
    });
  });

  describe('setAutoHeartbeatSettings', () => {
    it('persists settings to localStorage', () => {
      setAutoHeartbeatSettings({ deviceId: 3, intervalSeconds: 30 });
      expect(getAutoHeartbeatSettings()).toEqual({ deviceId: 3, intervalSeconds: 30 });
    });

    it(`dispatches ${SETTINGS_EVENT}`, () => {
      const handler = vi.fn();
      window.addEventListener(SETTINGS_EVENT, handler);
      setAutoHeartbeatSettings({ deviceId: 1, intervalSeconds: 60 });
      window.removeEventListener(SETTINGS_EVENT, handler);
      expect(handler).toHaveBeenCalledOnce();
    });
  });

  describe('clearAutoHeartbeatSettings', () => {
    it('removes key from localStorage', () => {
      setAutoHeartbeatSettings({ deviceId: 1, intervalSeconds: 60 });
      clearAutoHeartbeatSettings();
      expect(getAutoHeartbeatSettings()).toBeNull();
    });

    it(`dispatches ${SETTINGS_EVENT}`, () => {
      const handler = vi.fn();
      window.addEventListener(SETTINGS_EVENT, handler);
      clearAutoHeartbeatSettings();
      window.removeEventListener(SETTINGS_EVENT, handler);
      expect(handler).toHaveBeenCalledOnce();
    });
  });

  describe('storeClientIp / getStoredClientIp', () => {
    it('persists the IP', () => {
      storeClientIp('1.2.3.4');
      expect(getStoredClientIp()).toBe('1.2.3.4');
    });

    it(`dispatches ${CLIENT_IP_EVENT} with the IP as detail`, () => {
      const handler = vi.fn((e: Event) => (e as CustomEvent<string>).detail);
      window.addEventListener(CLIENT_IP_EVENT, handler);
      storeClientIp('5.6.7.8');
      window.removeEventListener(CLIENT_IP_EVENT, handler);
      expect(handler).toHaveBeenCalledOnce();
      expect((handler.mock.calls[0][0] as CustomEvent<string>).detail).toBe('5.6.7.8');
    });

    it('returns null when no IP stored', () => {
      expect(getStoredClientIp()).toBeNull();
    });
  });
});
