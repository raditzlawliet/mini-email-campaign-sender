<script>
    import { t } from "../i18n.svelte.js";

    let {
        provider = $bindable("smtp"),
        fromEmail = $bindable(""),
        smtpHost = $bindable(""),
        smtpPort = $bindable(""),
        smtpUsername = $bindable(""),
        smtpPassword = $bindable(""),
        smtpTLS = $bindable(false),
        smtpBatchSize = $bindable(50),
        sesRegion = $bindable(""),
        sesAccessKeyId = $bindable(""),
        sesSecretAccessKey = $bindable(""),
        sesUseTemplate = $bindable(false),
        sesTemplateName = $bindable(""),
        sesBatchSize = $bindable(50),
        disabled = false,
        onreset = () => {},
        onsave = () => {},
    } = $props();
</script>

<div>
    <fieldset class="fieldset">
        <label class="label" for="cfg-provider">{t("provider")}</label>
        <select
            id="cfg-provider"
            class="select w-full"
            bind:value={provider}
            {disabled}
        >
            <option value="smtp">{t("smtp")}</option>
            <option value="ses">{t("amazon_ses")}</option>
        </select>
    </fieldset>

    <fieldset class="fieldset">
        <label class="label" for="cfg-from">{t("from")}</label>
        <input
            id="cfg-from"
            type="text"
            class="input w-full"
            placeholder="sender@example.com"
            bind:value={fromEmail}
            {disabled}
        />
    </fieldset>

    {#if provider === "smtp"}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2">
            <fieldset class="fieldset">
                <label class="label" for="cfg-smtp-host">{t("host")}</label>
                <input
                    id="cfg-smtp-host"
                    type="text"
                    class="input w-full"
                    placeholder="smtp.example.com"
                    bind:value={smtpHost}
                    {disabled}
                />
            </fieldset>
            <fieldset class="fieldset">
                <label class="label" for="cfg-smtp-port">{t("port")}</label>
                <input
                    id="cfg-smtp-port"
                    type="text"
                    class="input w-full"
                    placeholder="587"
                    bind:value={smtpPort}
                    {disabled}
                />
            </fieldset>
            <fieldset class="fieldset">
                <label class="label" for="cfg-smtp-user">{t("username")}</label>
                <input
                    id="cfg-smtp-user"
                    type="text"
                    class="input w-full"
                    placeholder="user@example.com"
                    bind:value={smtpUsername}
                    {disabled}
                />
            </fieldset>
            <fieldset class="fieldset">
                <label class="label" for="cfg-smtp-pass">{t("password")}</label>
                <input
                    id="cfg-smtp-pass"
                    type="password"
                    class="input w-full"
                    placeholder="••••••••"
                    bind:value={smtpPassword}
                    {disabled}
                />
            </fieldset>
            <fieldset class="fieldset">
                <label class="label" for="cfg-smtp-batch"
                    >{t("batch_size")}</label
                >
                <input
                    id="cfg-smtp-batch"
                    type="number"
                    class="input w-full"
                    min="1"
                    max="500"
                    bind:value={smtpBatchSize}
                    {disabled}
                />
                <p class="fieldset-label">{t("emails_per_smtp")}</p>
            </fieldset>
        </div>
        <fieldset class="fieldset">
            <label class="label cursor-pointer justify-start gap-3 mt-2">
                <input
                    type="checkbox"
                    class="checkbox"
                    bind:checked={smtpTLS}
                    {disabled}
                />
                {t("enable_tls")}
            </label>
        </fieldset>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <fieldset class="fieldset">
                <label class="label" for="cfg-ses-region">{t("region")}</label>
                <input
                    id="cfg-ses-region"
                    type="text"
                    class="input w-full"
                    placeholder="us-east-1"
                    bind:value={sesRegion}
                    {disabled}
                />
            </fieldset>
            <fieldset class="fieldset">
                <label class="label" for="cfg-ses-key"
                    >{t("access_key_id")}</label
                >
                <input
                    id="cfg-ses-key"
                    type="text"
                    class="input w-full"
                    placeholder="AKIA..."
                    bind:value={sesAccessKeyId}
                    {disabled}
                />
            </fieldset>
        </div>
        <fieldset class="fieldset">
            <label class="label" for="cfg-ses-secret"
                >{t("secret_access_key")}</label
            >
            <input
                id="cfg-ses-secret"
                type="password"
                class="input w-full"
                placeholder="••••••••"
                bind:value={sesSecretAccessKey}
                {disabled}
            />
        </fieldset>

        <div class="divider my-2"></div>

        <fieldset class="fieldset">
            <label class="label cursor-pointer justify-start gap-3">
                <input
                    type="checkbox"
                    class="checkbox"
                    bind:checked={sesUseTemplate}
                    {disabled}
                />
                <span class="label-text">{t("use_ses_template")}</span>
            </label>
            <p class="fieldset-label text-base-content/50">
                {t("ses_template_desc")}
            </p>
        </fieldset>

        {#if sesUseTemplate}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4 mt-2">
                <fieldset class="fieldset">
                    <label class="label" for="cfg-ses-tpl-name"
                        >{t("template_name")}</label
                    >
                    <input
                        id="cfg-ses-tpl-name"
                        type="text"
                        class="input w-full"
                        placeholder="marketing-v2"
                        bind:value={sesTemplateName}
                        {disabled}
                    />
                </fieldset>
                <fieldset class="fieldset">
                    <label class="label" for="cfg-ses-batch"
                        >{t("batch_size")}</label
                    >
                    <input
                        id="cfg-ses-batch"
                        type="number"
                        class="input w-full"
                        min="1"
                        max="50"
                        bind:value={sesBatchSize}
                        {disabled}
                    />
                    <p class="fieldset-label">
                        {t("emails_per_ses")}
                    </p>
                </fieldset>
            </div>
        {/if}
    {/if}

    <div class="flex justify-end gap-2 mt-2">
        <button class="btn btn-ghost btn-sm" onclick={onreset} {disabled}
            >{t("reset_defaults")}</button
        >
        <button class="btn btn-outline btn-sm" onclick={onsave} {disabled}
            >{t("save_as_defaults")}</button
        >
    </div>
</div>
