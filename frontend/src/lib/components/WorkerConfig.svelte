<script>
    let {
        concurrency = $bindable(10),
        maxRetries = $bindable(3),
        batchSize = $bindable(50),
        backoffBase = $bindable("1s"),
        backoffMax = $bindable("30s"),
        disabled = false,
        onreset = () => {},
    } = $props();
</script>

<div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2">
        <fieldset class="fieldset">
            <label class="label" for="cfg-concurrency">Concurrency</label>
            <input
                id="cfg-concurrency"
                type="number"
                class="input w-full"
                min="1"
                max="100"
                bind:value={concurrency}
                {disabled}
            />
            <p class="fieldset-label">Number of parallel workers (1–100)</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-batchsize">SMTP Batch Size</label>
            <input
                id="cfg-batchsize"
                type="number"
                class="input w-full"
                min="1"
                max="500"
                bind:value={batchSize}
                {disabled}
            />
            <p class="fieldset-label">Emails per SMTP connection (1–500)</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-maxretries">Max Retries</label>
            <input
                id="cfg-maxretries"
                type="number"
                class="input w-full"
                min="0"
                max="10"
                bind:value={maxRetries}
                {disabled}
            />
            <p class="fieldset-label">Retry attempts on failure (0–10)</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-backoff-base">Backoff Base</label>
            <input
                id="cfg-backoff-base"
                type="text"
                class="input w-full"
                placeholder="1s"
                bind:value={backoffBase}
                {disabled}
            />
            <p class="fieldset-label">Initial retry delay, e.g. 1s, 500ms</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-backoff-max">Backoff Max</label>
            <input
                id="cfg-backoff-max"
                type="text"
                class="input w-full"
                placeholder="30s"
                bind:value={backoffMax}
                {disabled}
            />
            <p class="fieldset-label">Maximum retry delay cap</p>
        </fieldset>
    </div>
    <div class="flex justify-end">
        <button class="btn btn-ghost btn-sm" onclick={onreset} {disabled}
            >Reset defaults</button
        >
    </div>
</div>
