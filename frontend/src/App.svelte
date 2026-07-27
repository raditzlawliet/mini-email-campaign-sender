<script>
    import { onMount } from "svelte";
    import { Check, ChevronDown, ChevronUp } from "@lucide/svelte";
    import CampaignForm from "./lib/components/CampaignForm.svelte";

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
    let dropdownOpen = $state(false);

    function onThemeChange(e) {
        currentTheme = e.target.value;
    }

    onMount(async () => {
        // Read config theme and set the matching radio
        try {
            const res = await fetch("/api/campaign/config");
            const data = await res.json();
            const t = data?.app?.theme || "dark";
            currentTheme = t;
            const radio = document.querySelector(
                `input.theme-controller[value="${t}"]`,
            );
            if (radio) radio.checked = true;
        } catch {
            currentTheme = "dark";
        }
    });
</script>

<div class="min-h-screen bg-base-200">
    <div class="navbar bg-base-100 shadow-sm">
        <div class="navbar-start"></div>
        <div class="navbar-center">
            <span class="font-bold text-xl">Mini Email Campaign Sender</span>
        </div>
        <div class="navbar-end">
            <div
                class="dropdown dropdown-end"
                class:dropdown-open={dropdownOpen}
                onfocusin={() => (dropdownOpen = true)}
                onfocusout={() => (dropdownOpen = false)}
            >
                <button
                    tabindex="0"
                    class="btn btn-ghost btn-sm gap-2 capitalize"
                    onclick={() => (dropdownOpen = !dropdownOpen)}
                >
                    <div
                        class="bg-base-100 grid shrink-0 grid-cols-2 gap-0.5 rounded-md p-0.5 shadow-sm"
                    >
                        <div class="bg-base-content size-1 rounded-full"></div>
                        <div class="bg-primary size-1 rounded-full"></div>
                        <div class="bg-secondary size-1 rounded-full"></div>
                        <div class="bg-accent size-1 rounded-full"></div>
                    </div>
                    {#if dropdownOpen}
                        <ChevronUp class="w-4 h-4" />
                    {:else}
                        <ChevronDown class="w-4 h-4" />
                    {/if}
                </button>
                <fieldset
                    tabindex="-1"
                    class="dropdown-content bg-base-200 rounded-box z-1 w-60 p-2 shadow-2xl mt-2 max-h-80 overflow-y-auto flex flex-col gap-0.5"
                    onchange={onThemeChange}
                >
                    {#each THEMES as t}
                        <label
                            class="flex cursor-pointer items-center gap-3 px-2 py-1.5 rounded-btn hover:bg-base-300 capitalize"
                        >
                            <input
                                type="radio"
                                name="app-theme"
                                class="theme-controller hidden"
                                value={t}
                                checked={currentTheme === t}
                            />
                            <div
                                data-theme={t}
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
                            <div class="truncate">{t}</div>
                            <Check
                                class={currentTheme === t
                                    ? "ml-auto h-3 w-3 shrink-0"
                                    : "invisible ml-auto h-3 w-3 shrink-0"}
                            />
                        </label>
                    {/each}
                </fieldset>
            </div>
        </div>
    </div>

    <main class="container mx-auto max-w-4xl px-4 py-8">
        <CampaignForm />
    </main>
</div>
