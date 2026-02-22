<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api";

    let courses: any[] = $state([]);
    let enrollments: any[] = $state([]);
    let loading = $state(true);
    let error = $state("");

    onMount(async () => {
        try {
            const [courseRes, enrollRes] = await Promise.allSettled([
                api.courses.list(),
                api.enrollments.list(),
            ]);
            if (courseRes.status === "fulfilled")
                courses = courseRes.value.courses || [];
            if (enrollRes.status === "fulfilled")
                enrollments = enrollRes.value.enrollments || [];
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    });

    const stats = $derived([
        {
            label: "Total Courses",
            value: String(courses.length),
            icon: "📚",
            color: "from-primary-500 to-primary-600",
        },
        {
            label: "Enrolled",
            value: String(enrollments.length),
            icon: "🎓",
            color: "from-accent-500 to-accent-600",
        },
        {
            label: "Completion Rate",
            value:
                enrollments.length > 0
                    ? Math.round(
                          (enrollments.filter(
                              (e: any) => e.status === "completed",
                          ).length /
                              enrollments.length) *
                              100,
                      ) + "%"
                    : "—",
            icon: "✅",
            color: "from-success-500 to-green-600",
        },
        {
            label: "Published",
            value: String(
                courses.filter((c: any) => c.status === "published").length,
            ),
            icon: "⚡",
            color: "from-warning-500 to-orange-600",
        },
    ]);
</script>

<svelte:head>
    <title>Dashboard — LearnHub</title>
</svelte:head>

<div>
    <h1 class="text-2xl font-bold text-surface-100 mb-6">Dashboard</h1>

    {#if loading}
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            {#each Array(4) as _}
                <div
                    class="bg-surface-900/50 border border-surface-700/50 rounded-2xl p-5 animate-pulse"
                >
                    <div class="h-10 w-10 bg-surface-800 rounded-xl mb-3"></div>
                    <div class="h-6 w-16 bg-surface-800 rounded mb-1"></div>
                    <div class="h-4 w-24 bg-surface-800 rounded"></div>
                </div>
            {/each}
        </div>
    {:else if error}
        <div
            class="bg-error-500/10 border border-error-500/30 text-error-500 px-4 py-3 rounded-xl mb-6 text-sm"
        >
            {error}
        </div>
    {:else}
        <!-- Stats Grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            {#each stats as stat}
                <div
                    class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl p-5 hover:border-surface-700 transition-all group"
                >
                    <div class="flex items-center justify-between mb-3">
                        <span class="text-2xl">{stat.icon}</span>
                        <span
                            class="w-10 h-10 rounded-xl bg-gradient-to-br {stat.color} opacity-20 group-hover:opacity-30 transition-opacity"
                        ></span>
                    </div>
                    <p class="text-2xl font-bold text-surface-100">
                        {stat.value}
                    </p>
                    <p class="text-sm text-surface-200 mt-1">{stat.label}</p>
                </div>
            {/each}
        </div>

        <!-- Recent Courses -->
        <div
            class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl overflow-hidden"
        >
            <div
                class="p-5 border-b border-surface-700/50 flex items-center justify-between"
            >
                <h2 class="text-lg font-semibold text-surface-100">
                    Recent Courses
                </h2>
                <a
                    href="/dashboard/create"
                    class="text-sm text-primary-400 hover:text-primary-300 font-medium transition-colors"
                    >+ New Course</a
                >
            </div>
            {#if courses.length === 0}
                <div class="p-10 text-center text-surface-200">
                    No courses yet. Create your first course!
                </div>
            {:else}
                <div class="divide-y divide-surface-700/50">
                    {#each courses.slice(0, 5) as course}
                        <div
                            class="p-5 flex items-center gap-4 hover:bg-surface-800/30 transition-colors"
                        >
                            <div
                                class="w-12 h-12 rounded-xl bg-gradient-to-br from-primary-500/20 to-accent-500/20 flex items-center justify-center text-xl shrink-0"
                            >
                                📖
                            </div>
                            <div class="flex-1 min-w-0">
                                <p
                                    class="font-medium text-surface-100 truncate"
                                >
                                    {course.title}
                                </p>
                                <p class="text-sm text-surface-200">
                                    {course.category || "Uncategorized"}
                                </p>
                            </div>
                            <span
                                class="px-3 py-1 rounded-full text-xs font-medium {course.status ===
                                'published'
                                    ? 'bg-success-500/10 text-success-500'
                                    : 'bg-warning-500/10 text-warning-500'}"
                            >
                                {course.status}
                            </span>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    {/if}
</div>
