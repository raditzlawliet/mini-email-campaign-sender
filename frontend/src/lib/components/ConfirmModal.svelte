<script>
    import { t } from "../i18n.svelte.js";

    let {
        open = false,
        title = "",
        message = undefined,
        confirmLabel = undefined,
        cancelLabel = undefined,
        onconfirm = () => {},
        oncancel = () => {},
    } = $props();

    let resolvedMessage = $derived(message ?? t("default_confirm_msg"));
    let resolvedConfirm = $derived(confirmLabel ?? t("save"));
    let resolvedCancel = $derived(cancelLabel ?? t("cancel"));
</script>

{#if open}
    <dialog class="modal modal-open">
        <div class="modal-box">
            <h3 class="text-lg font-bold">{title}</h3>
            <p class="py-4 text-sm text-base-content/70">{resolvedMessage}</p>
            <div class="modal-action">
                <button class="btn btn-ghost btn-sm" onclick={oncancel}
                    >{resolvedCancel}</button
                >
                <button class="btn btn-primary btn-sm" onclick={onconfirm}
                    >{resolvedConfirm}</button
                >
            </div>
        </div>
        <form method="dialog" class="modal-backdrop">
            <button onclick={oncancel}>close</button>
        </form>
    </dialog>
{/if}
