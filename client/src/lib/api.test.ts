import { describe, it, expect } from 'vitest';

describe('API Client', () => {
    it('should export api object with required namespaces', async () => {
        const { api } = await import('./api');
        expect(api).toBeDefined();
        expect(api.auth).toBeDefined();
        expect(api.courses).toBeDefined();
        expect(api.enrollments).toBeDefined();
    });

    it('should have all auth methods', async () => {
        const { api } = await import('./api');
        expect(typeof api.auth.login).toBe('function');
        expect(typeof api.auth.register).toBe('function');
        expect(typeof api.auth.refresh).toBe('function');
        expect(typeof api.auth.logout).toBe('function');
        expect(typeof api.auth.me).toBe('function');
    });

    it('should have all courses methods', async () => {
        const { api } = await import('./api');
        expect(typeof api.courses.list).toBe('function');
        expect(typeof api.courses.get).toBe('function');
        expect(typeof api.courses.create).toBe('function');
        expect(typeof api.courses.addModule).toBe('function');
        expect(typeof api.courses.addLesson).toBe('function');
    });

    it('should have all enrollment methods', async () => {
        const { api } = await import('./api');
        expect(typeof api.enrollments.list).toBe('function');
        expect(typeof api.enrollments.enroll).toBe('function');
        expect(typeof api.enrollments.getProgress).toBe('function');
        expect(typeof api.enrollments.updateProgress).toBe('function');
    });

    it('should export default http instance', async () => {
        const mod = await import('./api');
        expect(mod.default).toBeDefined();
    });
});
