import { describe, expect, it } from 'vitest';
import { ROUTES, safeReturnTo } from '../routes';

describe('safeReturnTo', () => {
  it.each([
    '/devices',
    '/devices/owners/7?device=3&tab=rules',
    '/access-log?request=12#top',
  ])('keeps the in-app path %s', (path) => {
    expect(safeReturnTo(path)).toBe(path);
  });

  it.each([
    null,
    '',
    'devices',
    'https://evil.example/',
    '//evil.example',
    '///evil.example',
    '/\\evil.example',
    'javascript:alert(1)',
    '/ok\n//evil.example',
    '/ok\t',
  ])('falls back to the dashboard for %j', (raw) => {
    expect(safeReturnTo(raw)).toBe(ROUTES.dashboard);
  });
});
