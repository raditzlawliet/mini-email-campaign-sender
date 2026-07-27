<script>
    let {
        open = false,
        title = "",
        message = "This will write current settings to config.yaml.",
        confirmLabel = "Save",
        cancelLabel = "Cancel",
        onconfirm = () => {},
        oncancel = () => {},
    } = $props();
</script>

{#if open}
    <dialog class="modal modal-open">
        <div class="modal-box">
            <h3 class="text-lg font-bold">{title}</h3>
            <p class="py-4 text-sm text-base-content/70">{message}</p>
            <div class="modal-action">
                <button class="btn btn-ghost btn-sm" onclick={oncancel}
                    >{cancelLabel}</button
                >
                <button class="btn btn-primary btn-sm" onclick={onconfirm}
                    >{confirmLabel}</button
                >
            </div>
        </div>
        <form method="dialog" class="modal-backdrop">
            <button onclick={oncancel}>close</button>
        </form>
    </dialog>
{/if}
