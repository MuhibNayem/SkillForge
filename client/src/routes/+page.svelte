<svelte:head>
    <title>LearnHub — Modern Learning Management System</title>
</svelte:head>

<div class="min-h-screen bg-surface-950 flex flex-col">
    <!-- Hero -->
    <header class="relative overflow-hidden">
        <div
            class="absolute inset-0 bg-gradient-to-br from-primary-900/30 via-surface-950 to-accent-900/20"
        ></div>
        <div
            class="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[800px] bg-primary-500/5 rounded-full blur-3xl"
        ></div>

        <nav
            class="relative z-10 flex items-center justify-between px-6 lg:px-12 py-5"
        >
            <span
                class="text-2xl font-bold bg-gradient-to-r from-primary-400 to-accent-400 bg-clip-text text-transparent"
                >LearnHub</span
            >
            <div class="flex items-center gap-4">
                <a
                    href="/login"
                    class="px-5 py-2.5 text-surface-200 hover:text-surface-100 font-medium transition-colors text-sm"
                    >Sign In</a
                >
                <a
                    href="/register"
                    class="px-5 py-2.5 bg-primary-600 hover:bg-primary-500 text-white font-medium rounded-xl transition-colors text-sm"
                    >Get Started</a
                >
            </div>
        </nav>

        <div class="relative z-10 text-center px-6 pt-16 pb-24">
            <h1
                class="text-5xl lg:text-7xl font-bold text-surface-100 max-w-3xl mx-auto leading-tight"
            >
                Learn Without
                <span
                    class="bg-gradient-to-r from-primary-400 to-accent-400 bg-clip-text text-transparent"
                    >Limits</span
                >
            </h1>
            <p class="text-lg text-surface-200 mt-6 max-w-xl mx-auto">
                The modern LMS for teams that demand more. Create, deliver, and
                track learning at enterprise scale.
            </p>
            <div class="flex items-center justify-center gap-4 mt-8">
                <a
                    href="/register"
                    class="px-8 py-3.5 bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 text-white font-semibold rounded-xl shadow-lg shadow-primary-500/25 transition-all"
                >
                    Start Free Trial
                </a>
                <a
                    href="#features"
                    class="px-8 py-3.5 bg-surface-800/50 hover:bg-surface-800 text-surface-200 font-medium rounded-xl border border-surface-700 transition-all"
                >
                    See Features
                </a>
            </div>
        </div>
    </header>

    <!-- Features -->
    <section id="features" class="py-20 px-6 lg:px-12">
        <h2 class="text-3xl font-bold text-center text-surface-100 mb-12">
            Everything you need to teach and learn
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
            {#each [{ icon: "📚", title: "Course Builder", desc: "Drag-and-drop modules, video lessons, quizzes, and rich text — publish in minutes." }, { icon: "📊", title: "Analytics", desc: "Real-time dashboards for student progress, engagement, and completion rates." }, { icon: "🤖", title: "AI-Powered", desc: "Adaptive learning paths, smart recommendations, and AI quiz generation." }, { icon: "🔐", title: "Enterprise SSO", desc: "SAML 2.0, multi-tenancy, RBAC, and SOC 2-ready compliance controls." }, { icon: "🎓", title: "Certifications", desc: "Auto-generated certificates with unique verification codes." }, { icon: "⚡", title: "Microservices", desc: "Built on Go gRPC for sub-millisecond latency and infinite scalability." }] as feature}
                <div
                    class="bg-surface-900/50 backdrop-blur border border-surface-700/50 rounded-2xl p-6 hover:border-surface-700 hover:shadow-xl hover:shadow-primary-500/5 transition-all"
                >
                    <span class="text-3xl">{feature.icon}</span>
                    <h3 class="text-lg font-semibold text-surface-100 mt-4">
                        {feature.title}
                    </h3>
                    <p class="text-sm text-surface-200 mt-2">{feature.desc}</p>
                </div>
            {/each}
        </div>
    </section>

    <!-- Footer -->
    <footer
        class="border-t border-surface-700/50 px-6 py-8 text-center text-sm text-surface-200"
    >
        <p>&copy; 2026 LearnHub. Built with Go, Svelte 5, and ❤️</p>
    </footer>
</div>
