<script>
    import { t } from "../i18n.svelte.js";

    let {
        toField = $bindable("{name} <{email}>"),
        subject = $bindable(""),
        body = $bindable(""),
        useTemplate = false,
        disabled = false,
    } = $props();

    let _to = $state(toField);
    let _subject = $state(subject);
    let _body = $state(body);
    let bodyPreviewOpen = $state(false);
    let bodyPreviewHTML = $derived(`${_body}`);

    $effect(() => {
        _to = toField;
    });
    $effect(() => {
        _subject = subject;
    });
    $effect(() => {
        _body = body;
    });

    function flush() {
        toField = _to;
        subject = _subject;
        body = _body;
    }
</script>

<div class="card bg-base-100 shadow-sm">
    <div class="card-body">
        <h2 class="card-title text-lg">{t("email_template")}</h2>

        <fieldset class="fieldset">
            <label class="label" for="tpl-to">{t("to")}</label>
            <input
                id="tpl-to"
                type="text"
                class="input w-full font-mono text-sm"
                placeholder="{'{name}'} <{'{email}'}>"
                bind:value={_to}
                onblur={flush}
                {disabled}
            />
        </fieldset>

        {#if useTemplate}
            <div class="alert alert-info text-sm">
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    class="stroke-current shrink-0 w-5 h-5"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    ></path>
                </svg>
                <span>
                    {t("ses_template_info")}
                </span>
            </div>
        {:else}
            <fieldset class="fieldset">
                <label class="label" for="tpl-subject">{t("subject")}</label>
                <input
                    id="tpl-subject"
                    type="text"
                    class="input w-full"
                    placeholder="Hello {'{name}'}, welcome!"
                    bind:value={_subject}
                    onblur={flush}
                    {disabled}
                />
            </fieldset>

            <fieldset class="fieldset">
                <div class="flex items-center justify-between">
                    <label class="label" for="tpl-body">{t("body")}</label>
                    <button
                        class="btn btn-ghost btn-xs"
                        onclick={() => (bodyPreviewOpen = true)}
                        title={t("preview")}>{t("preview")}</button
                    >
                </div>
                <textarea
                    id="tpl-body"
                    class="textarea w-full font-mono text-sm"
                    rows="6"
                    placeholder="Hi {'{name}'},&#10;&#10;Thanks for joining.&#10;&#10;Best regards"
                    bind:value={_body}
                    onblur={flush}
                    {disabled}></textarea>
            </fieldset>
        {/if}
    </div>
</div>

<!-- Body preview modal -->
{#if !useTemplate && bodyPreviewOpen}
    <div class="modal modal-open" role="dialog" aria-modal="true">
        <div class="modal-box max-w-2xl w-11/12">
            <div class="flex items-center justify-between mb-4">
                <h3 class="text-lg font-bold">{t("body_preview")}</h3>
                <button
                    class="btn btn-sm btn-circle btn-ghost"
                    onclick={() => (bodyPreviewOpen = false)}
                    aria-label={t("close")}>✕</button
                >
            </div>
            {#if _body}
                <iframe
                    title={t("body_preview")}
                    sandbox="allow-popups"
                    srcdoc={bodyPreviewHTML}
                    class="w-full border border-base-300 rounded-btn"
                    style="height: 300px;"
                ></iframe>
            {:else}
                <div class="py-8 text-center text-base-content/50 italic">
                    {t("no_body_content")}
                </div>
            {/if}
            <div class="modal-action">
                <button class="btn" onclick={() => (bodyPreviewOpen = false)}
                    >{t("close")}</button
                >
            </div>
        </div>
        <div
            class="modal-backdrop"
            onclick={() => (bodyPreviewOpen = false)}
            onkeydown={(e) => e.key === "Escape" && (bodyPreviewOpen = false)}
            role="button"
            tabindex="0"
        >
            {t("close")}
        </div>
    </div>
{/if}
