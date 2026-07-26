<script>
    let {
        logToFile = $bindable(true),
        verbose = $bindable(false),
        disabled = false,
        onreset = () => {},
    } = $props();
</script>

<div>
    <div class="space-y-4">
        <fieldset class="fieldset">
            <label class="label cursor-pointer justify-start gap-3">
                <input
                    type="checkbox"
                    class="checkbox"
                    bind:checked={logToFile}
                    {disabled}
                />
                Log to file
            </label>
            <p class="fieldset-label text-xs">Write campaign events to logs/campaign_*.log</p>
        </fieldset>

        <fieldset class="fieldset">
            <label class="label cursor-pointer justify-start gap-3">
                <input
                    type="checkbox"
                    class="checkbox"
                    bind:checked={verbose}
                    {disabled}
                />
                Verbose logging
            </label>
            <p class="fieldset-label text-xs">Include per-email send/retry debug details in log and frontend</p>
        </fieldset>
    </div>
    <div class="flex justify-end">
        <button class="btn btn-ghost btn-sm" onclick={onreset} {disabled}
            >Reset defaults</button
        >
    </div>
</div>
