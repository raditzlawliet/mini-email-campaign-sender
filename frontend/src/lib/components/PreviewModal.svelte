<script>
    import { X, ChevronLeft, ChevronRight } from "@lucide/svelte";
    import { t } from "../i18n.svelte.js";

    let { open = false, previews = [], onclose = () => {} } = $props();

    let current = $state(0);
    let mode = $state("render"); // "render" | "code"

    $effect(() => {
        current = 0;
    });

    function prev() {
        if (current > 0) current--;
    }
    function next() {
        if (current < previews.length - 1) current++;
    }

    let preview = $derived(previews[current]);
    let renderHTML = $derived(preview?.body || "");
    let codeText = $derived(preview?.body || "");
</script>

{#if open}
    <div class="modal modal-open" role="dialog" aria-modal="true">
        <div class="modal-box max-w-3xl w-11/12 flex flex-col max-h-[90vh]">
            <div class="flex items-center justify-between mb-4 shrink-0">
                <h3 class="text-lg font-bold">{t("email_preview")}</h3>
                <button
                    class="btn btn-sm btn-circle btn-ghost"
                    onclick={onclose}
                    aria-label={t("close")}><X class="w-4 h-4" /></button
                >
            </div>

            {#if previews.length === 0}
                <div class="py-8 text-center text-base-content/50">
                    <p>{t("no_previews")}</p>
                    <p class="text-sm mt-1">
                        {t("click_preview")}
                    </p>
                </div>
            {:else}
                <!-- Navigation -->
                <div class="flex items-center justify-between mb-3 shrink-0">
                    <div class="flex items-center gap-2">
                        <button
                            class="btn btn-sm btn-outline"
                            onclick={prev}
                            disabled={current === 0}
                            ><ChevronLeft class="w-4 h-4" /></button
                        >
                        <span class="text-sm mx-1"
                            >{current + 1} of {previews.length}</span
                        >
                        <button
                            class="btn btn-sm btn-outline"
                            onclick={next}
                            disabled={current === previews.length - 1}
                            ><ChevronRight class="w-4 h-4" /></button
                        >
                    </div>
                    <div role="tablist" class="tabs tabs-bordered">
                        <button
                            role="tab"
                            class="tab {mode === 'render' ? 'tab-active' : ''}"
                            onclick={() => (mode = "render")}
                            >{t("render")}</button
                        >
                        <button
                            role="tab"
                            class="tab {mode === 'code' ? 'tab-active' : ''}"
                            onclick={() => (mode = "code")}>{t("code")}</button
                        >
                    </div>
                </div>

                <!-- Header info -->
                <div class="text-sm space-y-1 mb-3 shrink-0">
                    <div>
                        <span class="text-base-content/50"
                            >{t("from_label")}</span
                        > sender@example.com
                    </div>
                    <div>
                        <span class="text-base-content/50">{t("to_label")}</span
                        >
                        {preview.to}
                    </div>
                    <div>
                        <span class="text-base-content/50"
                            >{t("subject_label")}</span
                        >
                        {preview.subject}
                    </div>
                </div>

                {#if mode === "render"}
                    <div class="flex-1 min-h-0">
                        <iframe
                            title={t("email_preview")}
                            sandbox="allow-popups"
                            srcdoc={renderHTML}
                            class="w-full border border-base-300 rounded-box flex-1"
                            style="height: 400px;"
                        ></iframe>
                    </div>
                {:else}
                    <div
                        class="mockup-code text-sm overflow-y-auto flex-1 min-h-0 w-full"
                        style="max-height: 400px;"
                    >
                        {#each codeText.split("\n") as line}
                            <pre><code>{line}</code></pre>
                        {/each}
                    </div>
                {/if}
            {/if}

            <div class="modal-action shrink-0">
                <button class="btn" onclick={onclose}>{t("close")}</button>
            </div>
        </div>

        <div
            class="modal-backdrop"
            onclick={onclose}
            onkeydown={(e) => e.key === "Escape" && onclose()}
            role="button"
            tabindex="0"
        >
            {t("close")}
        </div>
    </div>
{/if}
