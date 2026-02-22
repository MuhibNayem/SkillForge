<script lang="ts">
    import { goto } from "$app/navigation";
    import { api } from "$lib/api";

    let firstName = $state("");
    let lastName = $state("");
    let email = $state("");
    let password = $state("");
    let role = $state("student");
    let error = $state("");
    let loading = $state(false);

    async function handleRegister() {
        error = "";
        loading = true;
        try {
            await api.auth.register({
                email,
                password,
                first_name: firstName,
                last_name: lastName,
            });
            goto("/login");
        } catch (e: any) {
            error =
                e.response?.data?.message || e.message || "Registration failed";
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>Register — LearnHub</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-surface-950 p-4">
    <div class="w-full max-w-md">
        <div class="text-center mb-8">
            <h1
                class="text-4xl font-bold bg-gradient-to-r from-primary-400 to-accent-400 bg-clip-text text-transparent"
            >
                LearnHub
            </h1>
            <p class="text-surface-200 mt-2">Create your account</p>
        </div>

        <form
            onsubmit={(e) => {
                e.preventDefault();
                handleRegister();
            }}
            class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl p-8 shadow-2xl shadow-primary-500/5"
        >
            {#if error}
                <div
                    class="bg-error-500/10 border border-error-500/30 text-error-500 px-4 py-3 rounded-lg mb-6 text-sm"
                >
                    {error}
                </div>
            {/if}

            <div class="grid grid-cols-2 gap-4 mb-5">
                <div>
                    <label
                        for="firstName"
                        class="block text-sm font-medium text-surface-200 mb-2"
                        >First Name</label
                    >
                    <input
                        id="firstName"
                        type="text"
                        bind:value={firstName}
                        required
                        class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-all"
                        placeholder="John"
                    />
                </div>
                <div>
                    <label
                        for="lastName"
                        class="block text-sm font-medium text-surface-200 mb-2"
                        >Last Name</label
                    >
                    <input
                        id="lastName"
                        type="text"
                        bind:value={lastName}
                        required
                        class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-all"
                        placeholder="Doe"
                    />
                </div>
            </div>

            <div class="mb-5">
                <label
                    for="regEmail"
                    class="block text-sm font-medium text-surface-200 mb-2"
                    >Email</label
                >
                <input
                    id="regEmail"
                    type="email"
                    bind:value={email}
                    required
                    class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-all"
                    placeholder="you@example.com"
                />
            </div>

            <div class="mb-5">
                <label
                    for="regPassword"
                    class="block text-sm font-medium text-surface-200 mb-2"
                    >Password</label
                >
                <input
                    id="regPassword"
                    type="password"
                    bind:value={password}
                    required
                    minlength="8"
                    class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-all"
                    placeholder="••••••••"
                />
            </div>

            <div class="mb-6">
                <label class="block text-sm font-medium text-surface-200 mb-2"
                    >I am a</label
                >
                <div class="grid grid-cols-2 gap-3">
                    <button
                        type="button"
                        onclick={() => (role = "student")}
                        class="py-3 px-4 rounded-xl border text-sm font-medium transition-all cursor-pointer {role ===
                        'student'
                            ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                            : 'border-surface-700 bg-surface-800/50 text-surface-200 hover:border-surface-200'}"
                    >
                        🎓 Student
                    </button>
                    <button
                        type="button"
                        onclick={() => (role = "instructor")}
                        class="py-3 px-4 rounded-xl border text-sm font-medium transition-all cursor-pointer {role ===
                        'instructor'
                            ? 'border-accent-500 bg-accent-500/10 text-accent-400'
                            : 'border-surface-700 bg-surface-800/50 text-surface-200 hover:border-surface-200'}"
                    >
                        📚 Instructor
                    </button>
                </div>
            </div>

            <button
                type="submit"
                disabled={loading}
                class="w-full py-3 px-4 bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 text-white font-semibold rounded-xl shadow-lg shadow-primary-500/25 transition-all duration-200 disabled:opacity-50 cursor-pointer"
            >
                {loading ? "Creating account..." : "Create Account"}
            </button>

            <p class="text-center text-sm text-surface-200 mt-6">
                Already have an account?
                <a
                    href="/login"
                    class="text-primary-400 hover:text-primary-300 font-medium transition-colors"
                    >Sign in</a
                >
            </p>
        </form>
    </div>
</div>
