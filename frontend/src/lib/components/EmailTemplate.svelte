<script>
    let {
        toField = $bindable("{name} <{email}>"),
        subject = $bindable(""),
        body = $bindable(""),
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
        <h2 class="card-title text-lg">Email Template</h2>

        <fieldset class="fieldset">
            <label class="label" for="tpl-to">To</label>
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

        <fieldset class="fieldset">
            <label class="label" for="tpl-subject">Subject</label>
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
                <label class="label" for="tpl-body">Body</label>
                <button
                    class="btn btn-ghost btn-xs"
                    onclick={() => (bodyPreviewOpen = true)}
                    title="Preview body">Preview</button
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
    </div>
</div>

<!-- Body preview modal -->
{#if bodyPreviewOpen}
    <div class="modal modal-open" role="dialog" aria-modal="true">
        <div class="modal-box max-w-2xl w-11/12">
            <div class="flex items-center justify-between mb-4">
                <h3 class="text-lg font-bold">Body Preview</h3>
                <button
                    class="btn btn-sm btn-circle btn-ghost"
                    onclick={() => (bodyPreviewOpen = false)}
                    aria-label="Close">✕</button
                >
            </div>
            {#if _body}
                <iframe
                    title="Email body preview"
                    sandbox="allow-popups"
                    srcdoc={bodyPreviewHTML}
                    class="w-full border border-base-300 rounded-btn"
                    style="height: 300px;"
                ></iframe>
            {:else}
                <div class="py-8 text-center text-base-content/50 italic">
                    No body content to preview.
                </div>
            {/if}
            <div class="modal-action">
                <button class="btn" onclick={() => (bodyPreviewOpen = false)}
                    >Close</button
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
            Close
        </div>
    </div>
{/if}
