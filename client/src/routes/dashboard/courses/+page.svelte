<script lang="ts">
    import { onMount } from "svelte";
    import { api } from "$lib/api";

    let courses: any[] = $state([]);
    let loading = $state(true);
    let error = $state("");
    let enrolling = $state("");
    let search = $state("");
    let category = $state("all");

    onMount(async () => {
        try {
            const res = await api.courses.list();
            courses = res.courses || [];
        } catch (e: any) {
            error = e.message;
        } finally {
            loading = false;
        }
    });

    const categories = $derived([
        "all",
        ...new Set(courses.map((c: any) => c.category).filter(Boolean)),
    ]);

    const filteredCourses = $derived(
        courses.filter(
            (c: any) =>
                (category === "all" || c.category === category) &&
                c.title.toLowerCase().includes(search.toLowerCase()),
        ),
    );

    async function handleEnroll(courseId: string) {
        enrolling = courseId;
        try {
            await api.enrollments.enroll(courseId);
            alert("Enrolled successfully!");
        } catch (e: any) {
            alert("Enrollment failed: " + e.message);
        } finally {
            enrolling = "";
        }
    }
</script>

<svelte:head>
    <title>Course Catalog — LearnHub</title>
</svelte:head>

<div>
    <div
        class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6"
    >
        <h1 class="text-2xl font-bold text-surface-100">Course Catalog</h1>
        <input
            type="search"
            bind:value={search}
            placeholder="Search courses..."
            class="w-full sm:w-72 px-4 py-2.5 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 text-sm"
        />
    </div>

    <!-- Category Filters -->
    <div class="flex gap-2 mb-6 flex-wrap">
        {#each categories as cat}
            <button
                onclick={() => (category = cat)}
                class="px-4 py-2 rounded-xl text-sm font-medium transition-all cursor-pointer {category ===
                cat
                    ? 'bg-primary-500/20 text-primary-400 border border-primary-500/30'
                    : 'bg-surface-800/50 text-surface-200 border border-surface-700 hover:border-surface-200'}"
            >
                {cat === "all" ? "All" : cat}
            </button>
        {/each}
    </div>

    {#if loading}
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {#each Array(6) as _}
                <div
                    class="bg-surface-900/50 border border-surface-700/50 rounded-2xl overflow-hidden animate-pulse"
                >
                    <div class="h-36 bg-surface-800"></div>
                    <div class="p-5 space-y-3">
                        <div class="h-4 w-20 bg-surface-800 rounded"></div>
                        <div class="h-5 w-full bg-surface-800 rounded"></div>
                        <div
                            class="h-10 w-full bg-surface-800 rounded-xl"
                        ></div>
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
    {:else if filteredCourses.length === 0}
        <div class="text-center py-16">
            <p class="text-5xl mb-4">🔍</p>
            <p class="text-surface-200 text-lg">No courses found.</p>
        </div>
    {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {#each filteredCourses as course}
                <div
                    class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl overflow-hidden hover:border-surface-700 hover:shadow-xl hover:shadow-primary-500/5 transition-all group"
                >
                    <div
                        class="h-36 bg-gradient-to-br from-primary-500/10 to-accent-500/10 flex items-center justify-center text-5xl group-hover:scale-105 transition-transform"
                    >
                        📖
                    </div>
                    <div class="p-5">
                        <div class="flex items-center gap-2 mb-2">
                            <span
                                class="px-2 py-0.5 rounded-md bg-primary-500/10 text-primary-400 text-xs font-medium"
                                >{course.category || "General"}</span
                            >
                            <span
                                class="px-2 py-0.5 rounded-md bg-surface-800 text-surface-200 text-xs"
                                >{course.difficulty || "beginner"}</span
                            >
                        </div>
                        <h3
                            class="font-semibold text-surface-100 mb-2 line-clamp-2"
                        >
                            {course.title}
                        </h3>
                        <p class="text-sm text-surface-200 mb-3 line-clamp-2">
                            {course.description || ""}
                        </p>
                        <button
                            onclick={() => handleEnroll(course.course_id)}
                            disabled={enrolling === course.course_id}
                            class="mt-1 w-full py-2.5 bg-primary-600/20 hover:bg-primary-600/30 text-primary-400 font-medium rounded-xl transition-all text-sm cursor-pointer disabled:opacity-50"
                        >
                            {enrolling === course.course_id
                                ? "Enrolling..."
                                : "Enroll Now"}
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
