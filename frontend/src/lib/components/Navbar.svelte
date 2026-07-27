<script>
    import { onMount } from "svelte";
    import { ChevronDown, ChevronUp, GlobeIcon } from "@lucide/svelte";
    import { t, setLanguage, getLanguage, LANGS } from "../i18n.svelte.js";

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
        fetch("/api/config/save", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ app: { theme: t } }),
        }).catch(() => {});
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
        fetch("/api/config/save", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ app: { language: code } }),
        }).catch(() => {});
    }

    onMount(async () => {
        document.addEventListener("click", handleDocClick);
        try {
            const res = await fetch("/api/campaign/config");
            const data = await res.json();
            const t = data?.app?.theme || "dark";
            const l = data?.app?.language || "en";
            currentTheme = t;
            document.documentElement.dataset.theme = currentTheme;
            setLanguage(l);
        } catch {
            currentTheme = "dark";
            setLanguage("en");
        }

        return () => document.removeEventListener("click", handleDocClick);
    });
</script>

<div class="navbar bg-base-100 shadow-sm h-9">
    <div class="navbar-start"></div>
    <div class="navbar-center">
        <span class="font-bold text-xl">{t("app_title")}</span>
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
    </div>
</div>
