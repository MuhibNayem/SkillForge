<script lang="ts">
    import { goto } from "$app/navigation";
    import { auth, logout } from "$lib/stores/auth";
    let { children } = $props();
    let sidebarOpen = $state(true);

    async function handleLogout() {
        await logout();
        goto("/login");
    }
</script>

<div class="min-h-screen flex bg-surface-950">
    <!-- Sidebar -->
    <aside
        class="w-64 bg-surface-900/50 backdrop-blur border-r border-surface-700/50 flex flex-col shrink-0 {sidebarOpen
            ? ''
            : 'hidden'} transition-all"
    >
        <div class="p-6 border-b border-surface-700/50">
            <a
                href="/dashboard"
                class="text-2xl font-bold bg-gradient-to-r from-primary-400 to-accent-400 bg-clip-text text-transparent"
            >
                LearnHub
            </a>
        </div>

        <nav class="flex-1 p-4 space-y-1">
            <a
                href="/dashboard"
                class="flex items-center gap-3 px-4 py-3 rounded-xl text-surface-200 hover:bg-surface-800/50 hover:text-surface-100 transition-all text-sm font-medium"
            >
                <span>📊</span> Dashboard
            </a>
            <a
                href="/dashboard/courses"
                class="flex items-center gap-3 px-4 py-3 rounded-xl text-surface-200 hover:bg-surface-800/50 hover:text-surface-100 transition-all text-sm font-medium"
            >
                <span>📚</span> Courses
            </a>
            <a
                href="/dashboard/my-courses"
                class="flex items-center gap-3 px-4 py-3 rounded-xl text-surface-200 hover:bg-surface-800/50 hover:text-surface-100 transition-all text-sm font-medium"
            >
                <span>🎓</span> My Learning
            </a>
            <a
                href="/dashboard/create"
                class="flex items-center gap-3 px-4 py-3 rounded-xl text-surface-200 hover:bg-surface-800/50 hover:text-surface-100 transition-all text-sm font-medium"
            >
                <span>✏️</span> Create Course
            </a>
        </nav>

        <div class="p-4 border-t border-surface-700/50">
            <button
                onclick={handleLogout}
                class="flex items-center gap-3 px-4 py-3 rounded-xl text-surface-200 hover:bg-error-500/10 hover:text-error-500 transition-all text-sm font-medium w-full cursor-pointer"
            >
                <span>🚪</span> Sign Out
            </button>
        </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-auto">
        <header
            class="h-16 border-b border-surface-700/50 bg-surface-900/30 backdrop-blur flex items-center px-6 sticky top-0 z-10"
        >
            <button
                onclick={() => (sidebarOpen = !sidebarOpen)}
                class="mr-4 text-surface-200 hover:text-surface-100 transition-colors cursor-pointer"
            >
                ☰
            </button>
            <div class="flex-1"></div>
            <div
                class="w-9 h-9 rounded-full bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center text-white text-sm font-bold"
            >
                U
            </div>
        </header>
        <div class="p-6 lg:p-8">
            {@render children()}
        </div>
    </main>
</div>
