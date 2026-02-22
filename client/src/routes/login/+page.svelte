<script lang="ts">
    import { goto } from "$app/navigation";
    import { login } from "$lib/stores/auth";

    let email = $state("");
    let password = $state("");
    let error = $state("");
    let loading = $state(false);

    async function handleLogin() {
        error = "";
        loading = true;
        try {
            await login(email, password);
            goto("/dashboard");
        } catch (e: any) {
            error =
                e.response?.data?.message || e.message || "Invalid credentials";
        } finally {
            loading = false;
        }
    }
</script>

<svelte:head>
    <title>Login — LearnHub</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-surface-950 p-4">
    <div class="w-full max-w-md">
        <div class="text-center mb-8">
            <h1
                class="text-4xl font-bold bg-gradient-to-r from-primary-400 to-accent-400 bg-clip-text text-transparent"
            >
                LearnHub
            </h1>
            <p class="text-surface-200 mt-2">Sign in to your account</p>
        </div>

        <form
            onsubmit={(e) => {
                e.preventDefault();
                handleLogin();
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

            <div class="mb-5">
                <label
                    for="email"
                    class="block text-sm font-medium text-surface-200 mb-2"
                    >Email</label
                >
                <input
                    id="email"
                    type="email"
                    bind:value={email}
                    required
                    class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-all"
                    placeholder="you@example.com"
                />
            </div>

            <div class="mb-6">
                <label
                    for="password"
                    class="block text-sm font-medium text-surface-200 mb-2"
                    >Password</label
                >
                <input
                    id="password"
                    type="password"
                    bind:value={password}
                    required
                    class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-xl text-surface-100 placeholder-surface-200/50 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-all"
                    placeholder="••••••••"
                />
            </div>

            <button
                type="submit"
                disabled={loading}
                class="w-full py-3 px-4 bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 text-white font-semibold rounded-xl shadow-lg shadow-primary-500/25 transition-all duration-200 disabled:opacity-50 cursor-pointer"
            >
                {loading ? "Signing in..." : "Sign In"}
            </button>

            <p class="text-center text-sm text-surface-200 mt-6">
                Don't have an account?
                <a
                    href="/register"
                    class="text-primary-400 hover:text-primary-300 font-medium transition-colors"
                    >Sign up</a
                >
            </p>
        </form>
    </div>
</div>
