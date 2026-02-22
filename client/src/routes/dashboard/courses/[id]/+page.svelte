<script lang="ts">
    import { onMount } from "svelte";
    import { page } from "$app/state";
    import { api } from "$lib/api";

    let course: any = $state(null);
    let loading = $state(true);
    let error = $state("");
    let enrolling = $state(false);

    onMount(async () => {
        try {
            course = await api.courses.get(page.params.id);
        } catch (e: any) {
            error =
                e.response?.data?.message ||
                e.message ||
                "Failed to load course";
        } finally {
            loading = false;
        }
    });

    async function handleEnroll() {
        enrolling = true;
        try {
            await api.enrollments.enroll(page.params.id);
            alert("Enrolled successfully!");
        } catch (e: any) {
            alert(
                "Enrollment failed: " +
                    (e.response?.data?.message || e.message),
            );
        } finally {
            enrolling = false;
        }
    }
</script>

<svelte:head>
    <title>{course?.title || "Course"} — LearnHub</title>
</svelte:head>

<div>
    {#if loading}
        <div class="animate-pulse space-y-6">
            <div class="h-8 w-64 bg-surface-800 rounded"></div>
            <div class="h-4 w-full bg-surface-800 rounded"></div>
            <div class="h-48 bg-surface-800 rounded-2xl"></div>
        </div>
    {:else if error}
        <div
            class="bg-error-500/10 border border-error-500/30 text-error-500 px-4 py-3 rounded-xl text-sm"
        >
            {error}
        </div>
    {:else if course}
        <!-- Header -->
        <div class="flex flex-col md:flex-row gap-6 mb-8">
            <div class="flex-1">
                <div class="flex items-center gap-2 mb-3">
                    <span
                        class="px-3 py-1 rounded-lg bg-primary-500/10 text-primary-400 text-xs font-medium"
                        >{course.category || "General"}</span
                    >
                    <span
                        class="px-3 py-1 rounded-lg bg-surface-800 text-surface-200 text-xs"
                        >{course.difficulty || "beginner"}</span
                    >
                    <span
                        class="px-3 py-1 rounded-full text-xs font-medium {course.status ===
                        'published'
                            ? 'bg-success-500/10 text-success-500'
                            : 'bg-warning-500/10 text-warning-500'}"
                        >{course.status}</span
                    >
                </div>
                <h1 class="text-3xl font-bold text-surface-100 mb-3">
                    {course.title}
                </h1>
                <p class="text-surface-200 leading-relaxed mb-6">
                    {course.description || "No description available."}
                </p>
                <div class="flex gap-3">
                    <button
                        onclick={handleEnroll}
                        disabled={enrolling}
                        class="px-6 py-3 bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 text-white font-semibold rounded-xl shadow-lg shadow-primary-500/25 transition-all cursor-pointer disabled:opacity-50"
                    >
                        {enrolling ? "Enrolling..." : "Enroll Now"}
                    </button>
                    <a
                        href="/dashboard/courses/{page.params.id}/player"
                        class="px-6 py-3 bg-surface-800 hover:bg-surface-700 text-surface-200 font-medium rounded-xl transition-colors border border-surface-700"
                    >
                        ▶ Start Learning
                    </a>
                </div>
            </div>
            <div
                class="w-full md:w-80 h-48 rounded-2xl bg-gradient-to-br from-primary-500/10 to-accent-500/10 flex items-center justify-center text-6xl shrink-0"
            >
                📖
            </div>
        </div>

        <!-- Modules & Lessons -->
        {#if course.modules && course.modules.length > 0}
            <div
                class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl overflow-hidden"
            >
                <div class="p-5 border-b border-surface-700/50">
                    <h2 class="text-lg font-semibold text-surface-100">
                        Course Content
                    </h2>
                    <p class="text-sm text-surface-200 mt-1">
                        {course.modules.length} modules · {course.modules.reduce(
                            (t: number, m: any) => t + (m.lessons?.length || 0),
                            0,
                        )} lessons
                    </p>
                </div>
                <div class="divide-y divide-surface-700/50">
                    {#each course.modules as mod, mi}
                        <div class="p-5">
                            <h3 class="font-medium text-surface-100 mb-3">
                                <span class="text-primary-400 mr-2"
                                    >Module {mi + 1}:</span
                                >
                                {mod.title}
                            </h3>
                            {#if mod.lessons && mod.lessons.length > 0}
                                <div class="ml-4 space-y-2">
                                    {#each mod.lessons as lesson, li}
                                        <div
                                            class="flex items-center gap-3 py-2 text-sm"
                                        >
                                            <span class="text-surface-200">
                                                {lesson.type === "video"
                                                    ? "📹"
                                                    : lesson.type === "quiz"
                                                      ? "❓"
                                                      : "📝"}
                                            </span>
                                            <span
                                                class="text-surface-200 flex-1"
                                                >{lesson.title}</span
                                            >
                                            <span
                                                class="text-xs text-surface-200/60 capitalize"
                                                >{lesson.type}</span
                                            >
                                        </div>
                                    {/each}
                                </div>
                            {:else}
                                <p class="text-sm text-surface-200/60 ml-4">
                                    No lessons yet.
                                </p>
                            {/if}
                        </div>
                    {/each}
                </div>
            </div>
        {:else}
            <div class="text-center py-12 text-surface-200">
                No course content available yet.
            </div>
        {/if}
    {/if}
</div>
