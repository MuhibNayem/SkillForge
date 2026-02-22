<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { page } from "$app/state";
    import { api } from "$lib/api";
    import Hls from "hls.js";
    import { marked } from "marked";

    let course: any = $state(null);
    let loading = $state(true);
    let error = $state("");
    let activeModuleIndex = $state(0);
    let activeLessonIndex = $state(0);
    let videoEl: HTMLVideoElement;
    let hlsInstance: Hls | null = null;
    let sidebarOpen = $state(true);

    const activeLesson = $derived(() => {
        if (!course?.modules?.[activeModuleIndex]?.lessons?.[activeLessonIndex])
            return null;
        return course.modules[activeModuleIndex].lessons[activeLessonIndex];
    });

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

    onDestroy(() => {
        if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
        }
    });

    function selectLesson(mi: number, li: number) {
        activeModuleIndex = mi;
        activeLessonIndex = li;
        if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
        }
    }

    function loadVideo(url: string) {
        if (!videoEl || !url) return;
        if (hlsInstance) hlsInstance.destroy();

        if (url.endsWith(".m3u8") && Hls.isSupported()) {
            hlsInstance = new Hls();
            hlsInstance.loadSource(url);
            hlsInstance.attachMedia(videoEl);
        } else {
            videoEl.src = url;
        }
    }

    function nextLesson() {
        if (!course?.modules) return;
        const mod = course.modules[activeModuleIndex];
        if (activeLessonIndex < (mod.lessons?.length || 0) - 1) {
            activeLessonIndex++;
        } else if (activeModuleIndex < course.modules.length - 1) {
            activeModuleIndex++;
            activeLessonIndex = 0;
        }
    }

    function prevLesson() {
        if (activeLessonIndex > 0) {
            activeLessonIndex--;
        } else if (activeModuleIndex > 0) {
            activeModuleIndex--;
            const mod = course.modules[activeModuleIndex];
            activeLessonIndex = (mod.lessons?.length || 1) - 1;
        }
    }

    function renderMarkdown(text: string): string {
        return marked.parse(text || "*No content available.*", {
            async: false,
        }) as string;
    }

    // Count total lessons
    function totalLessons(): number {
        if (!course?.modules) return 0;
        return course.modules.reduce(
            (t: number, m: any) => t + (m.lessons?.length || 0),
            0,
        );
    }

    function currentLessonNumber(): number {
        if (!course?.modules) return 0;
        let n = 0;
        for (let i = 0; i < activeModuleIndex; i++) {
            n += course.modules[i].lessons?.length || 0;
        }
        return n + activeLessonIndex + 1;
    }
</script>

<svelte:head>
    <title>{activeLesson()?.title || "Player"} — LearnHub</title>
</svelte:head>

<div class="flex h-[calc(100vh-4rem)] -m-6 lg:-m-8">
    {#if loading}
        <div class="flex-1 flex items-center justify-center">
            <div class="animate-pulse space-y-4 w-96">
                <div class="h-6 bg-surface-800 rounded w-48"></div>
                <div class="h-64 bg-surface-800 rounded-2xl"></div>
            </div>
        </div>
    {:else if error}
        <div class="flex-1 flex items-center justify-center">
            <div
                class="bg-error-500/10 border border-error-500/30 text-error-500 px-6 py-4 rounded-xl"
            >
                {error}
            </div>
        </div>
    {:else if course}
        <!-- Lesson Sidebar -->
        <aside
            class="w-80 bg-surface-900/50 border-r border-surface-700/50 overflow-y-auto shrink-0 {sidebarOpen
                ? ''
                : 'hidden'}"
        >
            <div class="p-4 border-b border-surface-700/50">
                <a
                    href="/dashboard/courses/{page.params.id}"
                    class="text-sm text-primary-400 hover:text-primary-300 transition-colors"
                    >← Back to course</a
                >
                <h2 class="font-semibold text-surface-100 mt-2 truncate">
                    {course.title}
                </h2>
                <p class="text-xs text-surface-200 mt-1">
                    Lesson {currentLessonNumber()} of {totalLessons()}
                </p>
            </div>
            <nav class="p-2">
                {#each course.modules || [] as mod, mi}
                    <div class="mb-2">
                        <p
                            class="px-3 py-2 text-xs font-semibold text-surface-200 uppercase tracking-wider"
                        >
                            Module {mi + 1}: {mod.title}
                        </p>
                        {#each mod.lessons || [] as lesson, li}
                            <button
                                onclick={() => selectLesson(mi, li)}
                                class="w-full text-left px-3 py-2.5 rounded-lg text-sm transition-all cursor-pointer flex items-center gap-2
                                    {activeModuleIndex === mi &&
                                activeLessonIndex === li
                                    ? 'bg-primary-500/10 text-primary-400 border-l-2 border-primary-500'
                                    : 'text-surface-200 hover:bg-surface-800/50 hover:text-surface-100'}"
                            >
                                <span class="text-xs shrink-0">
                                    {lesson.type === "video"
                                        ? "📹"
                                        : lesson.type === "quiz"
                                          ? "❓"
                                          : "📝"}
                                </span>
                                <span class="truncate">{lesson.title}</span>
                            </button>
                        {/each}
                    </div>
                {/each}
            </nav>
        </aside>

        <!-- Main Content Area -->
        <main class="flex-1 overflow-y-auto">
            <div class="p-6">
                <!-- Toggle Sidebar -->
                <button
                    onclick={() => (sidebarOpen = !sidebarOpen)}
                    class="mb-4 px-3 py-1.5 bg-surface-800/50 border border-surface-700 rounded-lg text-surface-200 text-sm hover:text-surface-100 transition-colors cursor-pointer"
                >
                    {sidebarOpen ? "◀ Hide" : "▶ Lessons"}
                </button>

                {#if activeLesson()}
                    <h2 class="text-xl font-bold text-surface-100 mb-4">
                        {activeLesson()?.title}
                    </h2>

                    {#if activeLesson()?.type === "video"}
                        <!-- Video Player -->
                        <div
                            class="relative bg-black rounded-2xl overflow-hidden mb-6 aspect-video"
                        >
                            <video
                                bind:this={videoEl}
                                controls
                                class="w-full h-full"
                                poster=""
                            >
                                <track kind="captions" />
                                Your browser does not support video playback.
                            </video>
                            {#if !activeLesson()?.content_url}
                                <div
                                    class="absolute inset-0 flex items-center justify-center bg-surface-900/80"
                                >
                                    <div class="text-center">
                                        <p class="text-4xl mb-3">📹</p>
                                        <p class="text-surface-200">
                                            Video content not yet uploaded.
                                        </p>
                                        <p
                                            class="text-xs text-surface-200/60 mt-1"
                                        >
                                            Upload via the Content Service API
                                        </p>
                                    </div>
                                </div>
                            {/if}
                        </div>
                    {:else if activeLesson()?.type === "text"}
                        <!-- Markdown Content -->
                        <div
                            class="bg-surface-900/50 border border-surface-700/50 rounded-2xl p-6 mb-6 prose prose-invert max-w-none
                            prose-headings:text-surface-100 prose-p:text-surface-200 prose-a:text-primary-400 prose-strong:text-surface-100
                            prose-code:bg-surface-800 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:text-primary-300
                            prose-pre:bg-surface-800 prose-pre:border prose-pre:border-surface-700"
                        >
                            {@html renderMarkdown(
                                activeLesson()?.content ||
                                    "# " +
                                        activeLesson()?.title +
                                        "\n\nContent will be available soon.",
                            )}
                        </div>
                    {:else}
                        <!-- Quiz Placeholder -->
                        <div
                            class="bg-surface-900/50 border border-surface-700/50 rounded-2xl p-8 mb-6 text-center"
                        >
                            <p class="text-4xl mb-3">❓</p>
                            <h3
                                class="text-lg font-semibold text-surface-100 mb-2"
                            >
                                Quiz: {activeLesson()?.title}
                            </h3>
                            <p class="text-surface-200">
                                Quiz functionality coming in Phase 2.
                            </p>
                        </div>
                    {/if}

                    <!-- Navigation -->
                    <div
                        class="flex items-center justify-between py-4 border-t border-surface-700/50"
                    >
                        <button
                            onclick={prevLesson}
                            disabled={activeModuleIndex === 0 &&
                                activeLessonIndex === 0}
                            class="px-5 py-2.5 bg-surface-800 hover:bg-surface-700 text-surface-200 text-sm font-medium rounded-xl transition-colors border border-surface-700 disabled:opacity-30 cursor-pointer"
                        >
                            ← Previous
                        </button>
                        <span class="text-sm text-surface-200"
                            >{currentLessonNumber()} / {totalLessons()}</span
                        >
                        <button
                            onclick={nextLesson}
                            disabled={activeModuleIndex ===
                                (course.modules?.length || 1) - 1 &&
                                activeLessonIndex ===
                                    (course.modules?.[activeModuleIndex]
                                        ?.lessons?.length || 1) -
                                        1}
                            class="px-5 py-2.5 bg-primary-600 hover:bg-primary-500 text-white text-sm font-medium rounded-xl transition-colors shadow-lg shadow-primary-500/25 disabled:opacity-30 cursor-pointer"
                        >
                            Next →
                        </button>
                    </div>
                {:else}
                    <div class="text-center py-16">
                        <p class="text-5xl mb-4">📚</p>
                        <p class="text-surface-200 text-lg">
                            No lessons available. Select a lesson from the
                            sidebar.
                        </p>
                    </div>
                {/if}
            </div>
        </main>
    {/if}
</div>
