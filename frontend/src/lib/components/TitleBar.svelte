<script>
    import { onMount } from "svelte";
    import {
        Mail,
        Minus,
        Square,
        Copy,
        X,
        GlobeIcon,
        ChevronDown,
        ChevronUp,
    } from "@lucide/svelte";
    import {
        WindowMinimise,
        WindowToggleMaximise,
        WindowIsMaximised,
        Quit,
        EventsOn,
        EventsOff,
        BrowserOpenURL,
    } from "../wailsjs/runtime/runtime";
    import {
        GetVersion,
        GetCampaignConfig,
        SaveConfig,
    } from "../wailsjs/go/app/App";
    import { t, setLanguage, getLanguage, LANGS } from "../i18n.svelte.js";
    import { app } from "../wailsjs/go/models";

    const REPO_URL =
        "https://github.com/raditzlawliet/mini-email-campaign-sender";

    const THEMES = [
        "dark",
        "light",
        "cupcake",
        "bumblebee",
        "emerald",
        "corporate",
        "synthwave",
        "retro",
        "cyberpunk",
        "valentine",
        "halloween",
        "garden",
        "forest",
        "aqua",
        "lofi",
        "pastel",
        "fantasy",
        "wireframe",
        "black",
        "luxury",
        "dracula",
        "cmyk",
        "autumn",
        "business",
        "acid",
        "lemonade",
        "night",
        "coffee",
        "winter",
        "dim",
        "nord",
        "sunset",
    ];

    let isMaximized = $state(false);
    let appVersion = $state("dev");
    let currentTheme = $state("dark");
    let themeDropdownOpen = $state(false);
    let langDropdownOpen = $state(false);
    let langRef = $state(null);
    let themeRef = $state(null);

    async function refreshMaximized() {
        try {
            isMaximized = await WindowIsMaximised();
        } catch {
            // runtime unavailable (plain browser dev) - keep current state
        }
    }

    function minimize() {
        WindowMinimise().catch(() => {});
    }

    async function toggleMaximize() {
        try {
            WindowToggleMaximise();
        } catch {
            // runtime unavailable (plain browser dev) - keep current state
        }
        refreshMaximized();
    }

    function closeApp() {
        Quit().catch(() => {});
    }

    function toggleLang() {
        if (langDropdownOpen) {
            langDropdownOpen = false;
        } else {
            themeDropdownOpen = false;
            langDropdownOpen = true;
        }
    }

    function toggleTheme() {
        if (themeDropdownOpen) {
            themeDropdownOpen = false;
        } else {
            langDropdownOpen = false;
            themeDropdownOpen = true;
        }
    }

    function handleDocClick(e) {
        const path = e.composedPath();
        if (langRef && !path.includes(langRef)) langDropdownOpen = false;
        if (themeRef && !path.includes(themeRef)) themeDropdownOpen = false;
    }

    async function persistTheme(theme) {
        SaveConfig(JSON.stringify({ app: { theme } })).catch(() => {});
    }

    function selectTheme(theme) {
        currentTheme = theme;
        document.documentElement.dataset.theme = theme;
        themeDropdownOpen = false;
        persistTheme(theme);
    }

    function selectLanguage(code) {
        setLanguage(code);
        langDropdownOpen = false;
        SaveConfig(JSON.stringify({ app: { language: code } })).catch(() => {});
    }

    function openExternal(url) {
        BrowserOpenURL(url);
    }

    onMount(() => {
        refreshMaximized();
        EventsOn("wails:maximised", () => {
            isMaximized = true;
        });
        EventsOn("wails:unmaximised", () => {
            isMaximized = false;
        });
        document.addEventListener("click", handleDocClick);

        (async () => {
            try {
                const data = await GetCampaignConfig();
                const theme = data?.app?.theme || "dark";
                const lang = data?.app?.language || "en";
                currentTheme = theme;
                document.documentElement.dataset.theme = currentTheme;
                setLanguage(lang);
            } catch {
                currentTheme = "dark";
                setLanguage("en");
            }
            try {
                appVersion = (await GetVersion()) || "dev";
            } catch {
                appVersion = "dev";
            }
        })();

        return () => {
            EventsOff("wails:maximised");
            EventsOff("wails:unmaximised");
            document.removeEventListener("click", handleDocClick);
        };
    });
</script>

<header
    dir="ltr"
    class="fixed inset-x-0 top-0 z-50 flex h-8 select-none items-stretch bg-base-100 shadow-sm"
>
    <!-- Drag region: title + version -->
    <div
        data-wails-drag
        role="button"
        tabindex="-1"
        ondblclick={toggleMaximize}
        class="flex min-w-0 flex-1 cursor-default items-center px-2"
    >
        <Mail class="size-4 shrink-0 text-primary me-2"></Mail>
        <span class="truncate text-sm font-bold tracking-wide"
            >{t("window_title")}</span
        >
        <button
            style="--wails-draggable: no-drag"
            onclick={() => openExternal(REPO_URL + "/releases")}
            class="btn btn-ghost btn-xs shrink-0 font-mono text-xs opacity-60 hover:opacity-100"
            title={t("view_releases")}
        >
            {appVersion}
        </button>
    </div>

    <!-- App controls + window controls -->
    <div class="flex items-stretch">
        <!-- Language picker -->
        <div
            class="dropdown dropdown-end dropdown-bottom"
            class:dropdown-open={langDropdownOpen}
            bind:this={langRef}
        >
            <button
                class="btn btn-ghost btn-sm gap-1 px-2 font-bold"
                aria-label={t("change_language")}
                title={t("change_language")}
                onclick={toggleLang}
            >
                <GlobeIcon class="size-4"></GlobeIcon>
                {#if langDropdownOpen}
                    <ChevronUp class="size-3.5" />
                {:else}
                    <ChevronDown class="size-3.5" />
                {/if}
            </button>
            {#if langDropdownOpen}
                <div
                    class="dropdown-content bg-base-200 rounded-box z-1 w-56 shadow-2xl max-h-80 overflow-y-auto"
                >
                    <ul class="menu menu-sm w-full">
                        {#each LANGS as l}
                            <li>
                                <button
                                    class={getLanguage() === l.code
                                        ? "menu-active"
                                        : ""}
                                    onclick={() => selectLanguage(l.code)}
                                >
                                    <span
                                        class="font-mono text-[.5625rem] font-bold tracking-[0.09375rem] opacity-40"
                                        >{l.code.toUpperCase()}</span
                                    >
                                    <span class="font-[sans-serif]"
                                        >{l.label}</span
                                    >
                                </button>
                            </li>
                        {/each}
                    </ul>
                </div>
            {/if}
        </div>

        <!-- Theme picker -->
        <div
            class="dropdown dropdown-end dropdown-bottom"
            class:dropdown-open={themeDropdownOpen}
            bind:this={themeRef}
        >
            <button
                class="btn btn-ghost btn-sm gap-2 px-2 capitalize"
                onclick={toggleTheme}
            >
                <div
                    class="bg-base-100 grid shrink-0 grid-cols-2 gap-0.5 rounded-md shadow-sm"
                >
                    <div class="bg-base-content size-1 rounded-full"></div>
                    <div class="bg-primary size-1 rounded-full"></div>
                    <div class="bg-secondary size-1 rounded-full"></div>
                    <div class="bg-accent size-1 rounded-full"></div>
                </div>
                {#if themeDropdownOpen}
                    <ChevronUp class="size-3.5" />
                {:else}
                    <ChevronDown class="size-3.5" />
                {/if}
            </button>
            {#if themeDropdownOpen}
                <div
                    class="dropdown-content bg-base-200 rounded-box z-1 w-48 shadow-2xl max-h-80 overflow-y-auto"
                >
                    <ul class="menu menu-sm w-full">
                        {#each THEMES as tm}
                            <li>
                                <button
                                    class={currentTheme === tm
                                        ? "menu-active capitalize"
                                        : "capitalize"}
                                    onclick={() => selectTheme(tm)}
                                >
                                    <div
                                        data-theme={tm}
                                        class="bg-base-100 grid shrink-0 grid-cols-2 gap-0.5 rounded-md p-1 shadow-sm"
                                    >
                                        <div
                                            class="bg-base-content size-1 rounded-full"
                                        ></div>
                                        <div
                                            class="bg-primary size-1 rounded-full"
                                        ></div>
                                        <div
                                            class="bg-secondary size-1 rounded-full"
                                        ></div>
                                        <div
                                            class="bg-accent size-1 rounded-full"
                                        ></div>
                                    </div>
                                    <span>{tm}</span>
                                </button>
                            </li>
                        {/each}
                    </ul>
                </div>
            {/if}
        </div>

        <!-- GitHub link -->
        <button
            onclick={() => openExternal(REPO_URL)}
            class="btn btn-ghost btn-sm px-2"
            title={t("github_repository")}
            aria-label={t("github_repository")}
        >
            <svg
                class="size-4"
                viewBox="0 0 24 24"
                fill="currentColor"
                xmlns="http://www.w3.org/2000/svg"
            >
                <path
                    d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12Z"
                />
            </svg>
        </button>

        <div class="my-1.5 w-px bg-base-content/10"></div>

        <!-- Window controls -->
        <div class="flex">
            <button
                class="flex w-10 items-center justify-center hover:bg-base-content/10"
                aria-label={t("window_minimize")}
                title={t("window_minimize")}
                onclick={minimize}
            >
                <Minus class="size-4"></Minus>
            </button>
            <button
                class="flex w-10 items-center justify-center hover:bg-base-content/10"
                aria-label={isMaximized
                    ? t("window_restore")
                    : t("window_maximize")}
                title={isMaximized ? t("window_restore") : t("window_maximize")}
                onclick={toggleMaximize}
            >
                {#if isMaximized}
                    <Copy class="size-3.5"></Copy>
                {:else}
                    <Square class="size-3.5"></Square>
                {/if}
            </button>
            <button
                class="flex w-10 items-center justify-center hover:bg-error hover:text-error-content"
                aria-label={t("window_close")}
                title={t("window_close")}
                onclick={closeApp}
            >
                <X class="size-4"></X>
            </button>
        </div>
    </div>
</header>
