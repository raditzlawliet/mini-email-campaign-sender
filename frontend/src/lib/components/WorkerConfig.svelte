<script>
    import { t } from "../i18n.svelte.js";

    let {
        concurrency = $bindable(10),
        maxRetries = $bindable(3),
        backoffBase = $bindable("1s"),
        backoffMax = $bindable("30s"),
        disabled = false,
        onreset = () => {},
        onsave = () => {},
    } = $props();
</script>

<div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2">
        <fieldset class="fieldset">
            <label class="label" for="cfg-concurrency">{t("concurrency")}</label
            >
            <input
                id="cfg-concurrency"
                type="number"
                class="input w-full"
                min="1"
                max="100"
                bind:value={concurrency}
                {disabled}
            />
            <p class="fieldset-label">{t("concurrency_desc")}</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-maxretries">{t("max_retries")}</label>
            <input
                id="cfg-maxretries"
                type="number"
                class="input w-full"
                min="0"
                max="10"
                bind:value={maxRetries}
                {disabled}
            />
            <p class="fieldset-label">{t("max_retries_desc")}</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-backoff-base"
                >{t("backoff_base")}</label
            >
            <input
                id="cfg-backoff-base"
                type="text"
                class="input w-full"
                placeholder="1s"
                bind:value={backoffBase}
                {disabled}
            />
            <p class="fieldset-label">{t("backoff_base_desc")}</p>
        </fieldset>
        <fieldset class="fieldset">
            <label class="label" for="cfg-backoff-max">{t("backoff_max")}</label
            >
            <input
                id="cfg-backoff-max"
                type="text"
                class="input w-full"
                placeholder="30s"
                bind:value={backoffMax}
                {disabled}
            />
            <p class="fieldset-label">{t("backoff_max_desc")}</p>
        </fieldset>
    </div>
    <div class="flex justify-end gap-2">
        <button class="btn btn-ghost btn-sm" onclick={onreset} {disabled}
            >{t("reset_defaults")}</button
        >
        <button class="btn btn-outline btn-sm" onclick={onsave} {disabled}
            >{t("save_as_defaults")}</button
        >
    </div>
</div>
