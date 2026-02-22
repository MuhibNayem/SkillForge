<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api";

    let enrollments: any[] = $state([]);
    let loading = $state(true);
    let error = $state("");

    onMount(async () => {
        try {
            const res = await api.enrollments.list();
            enrollments = res.enrollments || [];
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    });
</script>

<svelte:head>
    <title>My Learning — LearnHub</title>
</svelte:head>

<div>
    <h1 class="text-2xl font-bold text-surface-100 mb-6">My Learning</h1>

    {#if loading}
        <div class="space-y-4">
            {#each Array(3) as _}
                <div
                    class="bg-surface-900/50 border border-surface-700/50 rounded-2xl p-5 animate-pulse"
                >
                    <div class="flex items-center gap-4">
                        <div
                            class="w-16 h-16 bg-surface-800 rounded-xl shrink-0"
                        ></div>
                        <div class="flex-1 space-y-2">
                            <div class="h-5 w-48 bg-surface-800 rounded"></div>
                            <div class="h-4 w-32 bg-surface-800 rounded"></div>
                            <div
                                class="h-2 w-full bg-surface-800 rounded-full"
                            ></div>
                        </div>
                    </div>
                </div>
            {/each}
        </div>
    {:else if error}
        <div
            class="bg-error-500/10 border border-error-500/30 text-error-500 px-4 py-3 rounded-xl text-sm"
        >
            {error}
        </div>
    {:else if enrollments.length === 0}
        <div class="text-center py-16">
            <p class="text-5xl mb-4">📚</p>
            <p class="text-surface-200 text-lg">
                You haven't enrolled in any courses yet.
            </p>
            <a
                href="/dashboard/courses"
                class="inline-block mt-4 px-6 py-3 bg-primary-600 hover:bg-primary-500 text-white font-medium rounded-xl transition-colors"
                >Browse Courses</a
            >
        </div>
    {:else}
        <div class="space-y-4">
            {#each enrollments as enrollment}
                <div
                    class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl p-5 hover:border-surface-700 transition-all"
                >
                    <div
                        class="flex flex-col sm:flex-row items-start sm:items-center gap-4"
                    >
                        <div
                            class="w-16 h-16 rounded-xl bg-gradient-to-br from-primary-500/20 to-accent-500/20 flex items-center justify-center text-2xl shrink-0"
                        >
                            {enrollment.status === "completed" ? "🎉" : "📖"}
                        </div>
                        <div class="flex-1 min-w-0">
                            <h3 class="font-semibold text-surface-100 truncate">
                                Course: {enrollment.course_id}
                            </h3>
                            <p class="text-sm text-surface-200 mt-1">
                                Status: {enrollment.status}
                                · Enrolled {new Date(
                                    enrollment.enrolled_at * 1000,
                                ).toLocaleDateString()}
                            </p>
                            <div class="mt-3 flex items-center gap-3">
                                <div
                                    class="flex-1 h-2 bg-surface-800 rounded-full overflow-hidden"
                                >
                                    <div
                                        class="h-full bg-gradient-to-r from-primary-500 to-accent-500 rounded-full transition-all"
                                        style="width: {enrollment.overall_progress ||
                                            0}%"
                                    ></div>
                                </div>
                                <span
                                    class="text-sm font-medium text-primary-400 shrink-0"
                                    >{enrollment.overall_progress || 0}%</span
                                >
                            </div>
                        </div>
                        <span
                            class="px-3 py-1 rounded-full text-xs font-medium {enrollment.status ===
                            'completed'
                                ? 'bg-success-500/10 text-success-500'
                                : 'bg-primary-500/10 text-primary-400'}"
                        >
                            {enrollment.status}
                        </span>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
