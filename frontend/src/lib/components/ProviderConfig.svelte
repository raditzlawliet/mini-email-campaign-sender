<script>
    let {
        provider = $bindable("smtp"),
        fromEmail = $bindable(""),
        smtpHost = $bindable(""),
        smtpPort = $bindable(""),
        smtpUsername = $bindable(""),
        smtpPassword = $bindable(""),
        smtpTLS = $bindable(false),
        sesRegion = $bindable(""),
        sesAccessKeyId = $bindable(""),
        sesSecretAccessKey = $bindable(""),
        disabled = false,
        onreset = () => {},
    } = $props();
</script>

<div>
    <fieldset class="fieldset">
        <label class="label" for="cfg-provider">Provider</label>
        <select
            id="cfg-provider"
            class="select w-full"
            bind:value={provider}
            {disabled}
        >
            <option value="smtp">SMTP</option>
            <option value="ses">Amazon SES</option>
        </select>
    </fieldset>

    <fieldset class="fieldset">
        <label class="label" for="cfg-from">From</label>
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
                <label class="label" for="cfg-smtp-host">Host</label>
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
                <label class="label" for="cfg-smtp-port">Port</label>
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
                <label class="label" for="cfg-smtp-user">Username</label>
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
                <label class="label" for="cfg-smtp-pass">Password</label>
                <input
                    id="cfg-smtp-pass"
                    type="password"
                    class="input w-full"
                    placeholder="••••••••"
                    bind:value={smtpPassword}
                    {disabled}
                />
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
                Enable TLS
            </label>
        </fieldset>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <fieldset class="fieldset">
                <label class="label" for="cfg-ses-region">Region</label>
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
                <label class="label" for="cfg-ses-key">Access Key ID</label>
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
            <label class="label" for="cfg-ses-secret">Secret Access Key</label>
            <input
                id="cfg-ses-secret"
                type="password"
                class="input w-full"
                placeholder="••••••••"
                bind:value={sesSecretAccessKey}
                {disabled}
            />
        </fieldset>
    {/if}

    <div class="flex justify-end mt-2">
        <button class="btn btn-ghost btn-sm" onclick={onreset} {disabled}
            >Reset to defaults</button
        >
    </div>
</div>
