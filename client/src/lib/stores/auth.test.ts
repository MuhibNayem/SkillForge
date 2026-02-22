import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

describe('Auth Store', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.resetModules();
    });

    it('should export auth store and functions', async () => {
        const mod = await import('./auth');
        expect(mod.auth).toBeDefined();
        expect(typeof mod.initAuth).toBe('function');
        expect(typeof mod.login).toBe('function');
        expect(typeof mod.logout).toBe('function');
    });

    it('should start with unauthenticated state', async () => {
        const { auth } = await import('./auth');
        const state = get(auth);
        expect(state.isAuthenticated).toBe(false);
        expect(state.user).toBeNull();
        expect(state.isLoading).toBe(true);
    });

    it('logout should reset auth state', async () => {
        const { auth, logout } = await import('./auth');
        auth.set({
            user: { userId: '1', email: 'test@test.com', role: 'student', tenantId: 't1' },
            isAuthenticated: true,
            isLoading: false,
        });
        await logout();
        const state = get(auth);
        expect(state.isAuthenticated).toBe(false);
        expect(state.user).toBeNull();
        expect(state.isLoading).toBe(false);
    });
});
