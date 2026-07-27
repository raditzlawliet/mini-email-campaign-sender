<script>
    import { t } from "../i18n.svelte.js";

    let {
        logToFile = $bindable(true),
        verbose = $bindable(false),
        disabled = false,
        onreset = () => {},
        onsave = () => {},
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
                {t("log_to_file")}
            </label>
            <p class="fieldset-label text-xs">{t("log_to_file_desc")}</p>
        </fieldset>

        <fieldset class="fieldset">
            <label class="label cursor-pointer justify-start gap-3">
                <input
                    type="checkbox"
                    class="checkbox"
                    bind:checked={verbose}
                    {disabled}
                />
                {t("verbose_logging")}
            </label>
            <p class="fieldset-label text-xs">{t("verbose_logging_desc")}</p>
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
