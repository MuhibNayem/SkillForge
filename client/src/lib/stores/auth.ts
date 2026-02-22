import { writable } from 'svelte/store';
import { api } from '$lib/api';

interface User {
    userId: string;
    email: string;
    role: 'student' | 'instructor' | 'tenant_admin' | 'super_admin';
    tenantId: string;
}

interface AuthState {
    user: User | null;
    isAuthenticated: boolean;
    isLoading: boolean;
}

const initial: AuthState = { user: null, isAuthenticated: false, isLoading: true };

export const auth = writable<AuthState>(initial);

// Called on app init — check if user has valid session cookie
export async function initAuth() {
    try {
        const data = await api.auth.me();
        auth.set({
            user: { userId: data.user_id, email: data.email, role: data.role as User['role'], tenantId: data.tenant_id },
            isAuthenticated: true,
            isLoading: false,
        });
    } catch {
        auth.set({ user: null, isAuthenticated: false, isLoading: false });
    }
}

export async function login(email: string, password: string) {
    const data = await api.auth.login(email, password);
    // Server sets httpOnly cookies — we just update local state
    auth.set({
        user: { userId: data.user_id, email: data.email, role: data.role as User['role'], tenantId: '' },
        isAuthenticated: true,
        isLoading: false,
    });
    return data;
}

export async function logout() {
    try {
        await api.auth.logout();
    } catch {
        // Ignore — clear state regardless
    }
    auth.set({ user: null, isAuthenticated: false, isLoading: false });
}
