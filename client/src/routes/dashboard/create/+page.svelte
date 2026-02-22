<script lang="ts">
    import { goto } from "$app/navigation";
    import { api } from "$lib/api";

    let title = $state("");
    let description = $state("");
    let category = $state("");
    let difficulty = $state("beginner");
    let modules: {
        title: string;
        lessons: { title: string; type: string }[];
    }[] = $state([
        {
            title: "Module 1: Getting Started",
            lessons: [{ title: "Introduction", type: "video" }],
        },
    ]);
    let loading = $state(false);
    let error = $state("");
    let success = $state(false);

    function addModule() {
        modules = [
            ...modules,
            { title: `Module ${modules.length + 1}`, lessons: [] },
        ];
    }

    function addLesson(moduleIndex: number) {
        modules[moduleIndex].lessons = [
            ...modules[moduleIndex].lessons,
            { title: "", type: "video" },
        ];
    }

    function removeModule(i: number) {
        modules = modules.filter((_, idx) => idx !== i);
    }

    async function handleSubmit() {
        if (!title.trim()) {
            error = "Course title is required";
            return;
        }
        loading = true;
        error = "";
        try {
            const courseRes = await api.courses.create({
                title,
                description,
                category,
                difficulty,
                tenant_id: "00000000-0000-0000-0000-000000000001",
                instructor_id: "00000000-0000-0000-0000-000000000001",
            });

            // Add modules and lessons via API
            const courseId = courseRes.course_id;
            for (let mi = 0; mi < modules.length; mi++) {
                const mod = modules[mi];
                if (!mod.title.trim()) continue;
                const modRes = await api.courses.addModule(courseId, {
                    title: mod.title,
                    order: mi + 1,
                });
                for (let li = 0; li < mod.lessons.length; li++) {
                    const lesson = mod.lessons[li];
                    if (!lesson.title.trim()) continue;
                    await api.courses.addLesson(courseId, modRes.module_id, {
                        title: lesson.title,
                        type: lesson.type,
                        order: li + 1,
                    });
                }
            }

            success = true;
            setTimeout(() => goto("/dashboard"), 2000);
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>Create Course — LearnHub</title>
</svelte:head>

<div class="max-w-3xl">
    <h1 class="text-2xl font-bold text-surface-100 mb-6">Create Course</h1>

    {#if success}
        <div
            class="bg-success-500/10 border border-success-500/30 text-success-500 px-4 py-3 rounded-xl mb-6 text-sm"
        >
            ✅ Course created successfully! Redirecting...
        </div>
    {/if}

    {#if error}
        <div
            class="bg-error-500/10 border border-error-500/30 text-error-500 px-4 py-3 rounded-xl mb-6 text-sm"
        >
            {error}
        </div>
    {/if}

    <form
        onsubmit={(e) => {
            e.preventDefault();
            handleSubmit();
        }}
        class="space-y-6"
    >
        <!-- Basic Info -->
        <div
            class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl p-6"
        >
            <h2 class="text-lg font-semibold text-surface-100 mb-4">
                Basic Information
            </h2>
            <div class="space-y-4">
                <div>
                    <label
                        for="courseTitle"
                        class="block text-sm font-medium text-surface-200 mb-2"
                        >Course Title</label
                    >
                    <input
                        id="courseTitle"
                        type="text"
                        bind:value={title}
                        required
                        class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 transition-all"
                        placeholder="e.g. Introduction to Machine Learning"
                    />
                </div>
                <div>
                    <label
                        for="courseDesc"
                        class="block text-sm font-medium text-surface-200 mb-2"
                        >Description</label
                    >
                    <textarea
                        id="courseDesc"
                        bind:value={description}
                        rows="3"
                        class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 transition-all resize-none"
                        placeholder="What will students learn?"
                    ></textarea>
                </div>
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label
                            for="courseCategory"
                            class="block text-sm font-medium text-surface-200 mb-2"
                            >Category</label
                        >
                        <input
                            id="courseCategory"
                            type="text"
                            bind:value={category}
                            class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 transition-all"
                            placeholder="e.g. AI/ML"
                        />
                    </div>
                    <div>
                        <label
                            for="courseDiff"
                            class="block text-sm font-medium text-surface-200 mb-2"
                            >Difficulty</label
                        >
                        <select
                            id="courseDiff"
                            bind:value={difficulty}
                            class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 focus:outline-none focus:ring-2 focus:ring-primary-500/50 transition-all"
                        >
                            <option value="beginner">Beginner</option>
                            <option value="intermediate">Intermediate</option>
                            <option value="advanced">Advanced</option>
                        </select>
                    </div>
                </div>
            </div>
        </div>

        <!-- Modules & Lessons -->
        <div
            class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl p-6"
        >
            <div class="flex items-center justify-between mb-4">
                <h2 class="text-lg font-semibold text-surface-100">
                    Modules & Lessons
                </h2>
                <button
                    type="button"
                    onclick={addModule}
                    class="px-4 py-2 bg-primary-600/20 hover:bg-primary-600/30 text-primary-400 text-sm font-medium rounded-xl transition-all cursor-pointer"
                >
                    + Add Module
                </button>
            </div>
            <div class="space-y-4">
                {#each modules as mod, mi}
                    <div class="border border-surface-700/50 rounded-xl p-4">
                        <div class="flex items-center gap-3 mb-3">
                            <span class="text-surface-200 cursor-grab">⠿</span>
                            <input
                                type="text"
                                bind:value={mod.title}
                                class="flex-1 px-3 py-2 bg-surface-800/50 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/50 transition-all"
                            />
                            <button
                                type="button"
                                onclick={() => removeModule(mi)}
                                class="text-error-500 hover:text-error-500/80 text-sm cursor-pointer"
                                >✕</button
                            >
                        </div>
                        <div class="ml-8 space-y-2">
                            {#each mod.lessons as lesson, li}
                                <div class="flex items-center gap-2">
                                    <input
                                        type="text"
                                        bind:value={lesson.title}
                                        placeholder="Lesson title"
                                        class="flex-1 px-3 py-2 bg-surface-800/30 border border-surface-700/50 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-primary-500/50 transition-all"
                                    />
                                    <select
                                        bind:value={lesson.type}
                                        class="px-3 py-2 bg-surface-800/30 border border-surface-700/50 rounded-lg text-surface-100 text-sm focus:outline-none"
                                    >
                                        <option value="video">📹 Video</option>
                                        <option value="text">📝 Text</option>
                                        <option value="quiz">❓ Quiz</option>
                                    </select>
                                </div>
                            {/each}
                            <button
                                type="button"
                                onclick={() => addLesson(mi)}
                                class="text-sm text-surface-200 hover:text-primary-400 transition-colors cursor-pointer"
                            >
                                + Add Lesson
                            </button>
                        </div>
                    </div>
                {/each}
            </div>
        </div>

        <div class="flex gap-3">
            <button
                type="submit"
                disabled={loading}
                class="px-8 py-3 bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 text-white font-semibold rounded-xl shadow-lg shadow-primary-500/25 transition-all cursor-pointer disabled:opacity-50"
            >
                {loading ? "Creating..." : "Create Course"}
            </button>
        </div>
    </form>
</div>
