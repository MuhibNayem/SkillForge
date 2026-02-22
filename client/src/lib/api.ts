import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';

const API_BASE = '/api/v1';

// Axios instance — cookies are sent automatically with withCredentials
const http = axios.create({
    baseURL: API_BASE,
    withCredentials: true,
    headers: { 'Content-Type': 'application/json' },
    timeout: 15000,
});

// ──── Token refresh queue ────
let isRefreshing = false;
let failedQueue: { resolve: (v: unknown) => void; reject: (e: unknown) => void }[] = [];

function processQueue(error: unknown) {
    failedQueue.forEach((p) => {
        if (error) p.reject(error);
        else p.resolve(undefined);
    });
    failedQueue = [];
}

// ──── Response interceptor — auto-refresh on 401 ────
http.interceptors.response.use(
    (res) => res,
    async (error: AxiosError) => {
        const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

        // If 401 and not already retrying and not the refresh endpoint itself
        if (error.response?.status === 401 && !originalRequest._retry && !originalRequest.url?.includes('/auth/refresh')) {
            if (isRefreshing) {
                // Queue this request until refresh completes
                return new Promise((resolve, reject) => {
                    failedQueue.push({ resolve, reject });
                }).then(() => http(originalRequest));
            }

            originalRequest._retry = true;
            isRefreshing = true;

            try {
                // Refresh token is in httpOnly cookie, server handles it
                await http.post('/auth/refresh');
                processQueue(null);
                // Retry original request with new cookie
                return http(originalRequest);
            } catch (refreshError) {
                processQueue(refreshError);
                // Redirect to login on refresh failure
                if (typeof window !== 'undefined') {
                    window.location.href = '/login';
                }
                return Promise.reject(refreshError);
            } finally {
                isRefreshing = false;
            }
        }

        return Promise.reject(error);
    },
);

// ──── API methods ────
export const api = {
    auth: {
        login: (email: string, password: string) =>
            http.post<{ user_id: string; email: string; role: string }>('/auth/login', { email, password }).then((r) => r.data),

        register: (data: { email: string; password: string; first_name: string; last_name: string }) =>
            http.post<{ user_id: string; message: string }>('/auth/register', data).then((r) => r.data),

        refresh: () => http.post('/auth/refresh').then((r) => r.data),

        logout: () => http.post('/auth/logout').then((r) => r.data),

        me: () => http.get<{ user_id: string; email: string; role: string; tenant_id: string }>('/auth/me').then((r) => r.data),
    },
    courses: {
        list: () => http.get<{ courses: any[]; total: number }>('/courses').then((r) => r.data),
        get: (id: string) => http.get(`/courses/${id}`).then((r) => r.data),
        create: (data: any) => http.post('/courses', data).then((r) => r.data),
        addModule: (courseId: string, data: { title: string; order: number }) =>
            http.post(`/courses/${courseId}/modules`, data).then((r) => r.data),
        addLesson: (courseId: string, moduleId: string, data: { title: string; type: string; order: number }) =>
            http.post(`/courses/${courseId}/modules/${moduleId}/lessons`, data).then((r) => r.data),
    },
    enrollments: {
        list: () => http.get<{ enrollments: any[]; total: number }>('/enrollments').then((r) => r.data),
        enroll: (courseId: string) => http.post('/enrollments', { course_id: courseId }).then((r) => r.data),
        getProgress: (id: string) => http.get(`/enrollments/${id}/progress`).then((r) => r.data),
        updateProgress: (id: string, lessonId: string, percent: number) =>
            http.put(`/enrollments/${id}/progress`, { lesson_id: lessonId, percent_complete: percent }).then((r) => r.data),
    },
    content: {
        upload: (data: { filename: string; content_type: string; size_bytes: number; tenant_id: string; course_id: string; uploaded_by: string }) =>
            http.post('/content/upload', data).then((r) => r.data),
        get: (id: string) => http.get(`/content/${id}`).then((r) => r.data),
        delete: (id: string) => http.delete(`/content/${id}`).then((r) => r.data),
        list: (tenantId: string, courseId?: string) =>
            http.get('/content', { params: { tenant_id: tenantId, course_id: courseId } }).then((r) => r.data),
    },
    tenant: {
        get: (id: string) => http.get(`/tenants/${id}`).then((r) => r.data),
        updateSettings: (id: string, data: { theme?: string; logo_url?: string }) =>
            http.put(`/tenants/${id}/settings`, data).then((r) => r.data),
    },
};

export default http;
