<script>
    import { onMount } from "svelte";
    import { ChevronDown, ChevronUp, GlobeIcon } from "@lucide/svelte";
    import { t, setLanguage, getLanguage, LANGS } from "../i18n.svelte.js";
    import { GetVersion, GetCampaignConfig, SaveConfig } from "../wailsjs/go/app/App";
    import { BrowserOpenURL } from "../wailsjs/runtime/runtime";

    // static for now, later will use cicd if needed
    const REPO_URL =
        "https://github.com/raditzlawliet/mini-email-campaign-sender";

    // fetched from Go binding to maintain single source of truth
    let appVersion = $state("dev");

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

    let currentTheme = $state("dark");
    let themeDropdownOpen = $state(false);
    let langDropdownOpen = $state(false);
    let langRef = $state(null);
    let themeRef = $state(null);

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

    async function persistTheme(t) {
        SaveConfig(JSON.stringify({ app: { theme: t } })).catch(() => {});
    }

    async function selectTheme(tm) {
        currentTheme = tm;
        document.documentElement.dataset.theme = tm;
        themeDropdownOpen = false;
        persistTheme(tm);
    }

    async function selectLanguage(code) {
        setLanguage(code);
        langDropdownOpen = false;
        SaveConfig(JSON.stringify({ app: { language: code } })).catch(() => {});
    }

    function openExternal(url) {
        BrowserOpenURL(url);
    }

    onMount(async () => {
        document.addEventListener("click", handleDocClick);
        try {
            const data = await GetCampaignConfig();
            const t = data?.app?.theme || "dark";
            const l = data?.app?.language || "en";
            currentTheme = t;
            document.documentElement.dataset.theme = currentTheme;
            setLanguage(l);
        } catch {
            currentTheme = "dark";
            setLanguage("en");
        }

        try {
            appVersion = (await GetVersion()) || "dev";
        } catch {
            appVersion = "dev";
        }

        return () => document.removeEventListener("click", handleDocClick);
    });
</script>

<div class="navbar bg-base-100 shadow-sm h-9">
    <div class="navbar-start"></div>
    <div class="navbar-center">
        <div class="flex items-baseline gap-1">
            <span class="font-bold text-xl">{t("app_title")}</span>
            <button
                onclick={() => openExternal(REPO_URL + "/releases")}
                class="btn btn-ghost btn-xs text-xs font-mono opacity-60 hover:opacity-100"
                title="View releases"
            >
                {appVersion}
            </button>
        </div>
    </div>
    <div class="navbar-end gap-0.5">
        <!-- Language picker -->
        <div
            class="dropdown dropdown-end"
            class:dropdown-open={langDropdownOpen}
            bind:this={langRef}
        >
            <button
                class="btn btn-sm btn-ghost gap-1 px-1.5 text-[.5625rem] font-bold"
                aria-label={t("change_language")}
                title={t("change_language")}
                onclick={toggleLang}
            >
                <GlobeIcon class="w-4 h-4"></GlobeIcon>
                {#if langDropdownOpen}
                    <ChevronUp class="w-4 h-4" />
                {:else}
                    <ChevronDown class="w-4 h-4" />
                {/if}
            </button>
            {#if langDropdownOpen}
                <div
                    class="dropdown-content bg-base-200 rounded-box z-1 w-56 shadow-2xl mt-1 max-h-80 overflow-y-auto"
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
            class="dropdown dropdown-end"
            class:dropdown-open={themeDropdownOpen}
            bind:this={themeRef}
        >
            <button
                class="btn btn-ghost btn-sm gap-2 capitalize"
                onclick={toggleTheme}
            >
                <div
                    class="bg-base-100 grid shrink-0 grid-cols-2 gap-0.5 rounded-md p-0.5 shadow-sm"
                >
                    <div class="bg-base-content size-1 rounded-full"></div>
                    <div class="bg-primary size-1 rounded-full"></div>
                    <div class="bg-secondary size-1 rounded-full"></div>
                    <div class="bg-accent size-1 rounded-full"></div>
                </div>
                {#if themeDropdownOpen}
                    <ChevronUp class="w-4 h-4" />
                {:else}
                    <ChevronDown class="w-4 h-4" />
                {/if}
            </button>
            {#if themeDropdownOpen}
                <div
                    class="dropdown-content bg-base-200 rounded-box z-1 w-48 shadow-2xl mt-1 max-h-80 overflow-y-auto"
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
            class="btn btn-ghost btn-sm"
            title="GitHub repository"
            aria-label="GitHub repository"
        >
            <svg
                class="w-4 h-4"
                viewBox="0 0 24 24"
                fill="currentColor"
                xmlns="http://www.w3.org/2000/svg"
            >
                <path
                    d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12Z"
                />
            </svg>
        </button>
    </div>
</div>
