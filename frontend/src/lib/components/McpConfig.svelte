<script>
    import { t } from "../i18n.svelte.js";

    let {
        enabled = $bindable(true),
        host = $bindable("127.0.0.1"),
        port = $bindable("18799"),
        token = $bindable(""),
        status = "stopped",
        addr = "",
        errorMsg = "",
        onsave = () => {},
        oncleartoken = () => {},
    } = $props();
</script>

<div>
    <div class="space-y-4">
        <fieldset class="fieldset">
            <label class="label cursor-pointer justify-start gap-3">
                <input
                    type="checkbox"
                    class="checkbox"
                    bind:checked={enabled}
                />
                {t("mcp_enabled")}
            </label>
            <p class="fieldset-label text-xs">{t("mcp_enabled_desc")}</p>
        </fieldset>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <fieldset class="fieldset">
                <label class="label" for="mcp-host">{t("mcp_host")}</label>
                <input
                    id="mcp-host"
                    class="input w-full"
                    bind:value={host}
                    placeholder="127.0.0.1"
                />
            </fieldset>
            <fieldset class="fieldset">
                <label class="label" for="mcp-port">{t("mcp_port")}</label>
                <input
                    id="mcp-port"
                    class="input w-full"
                    type="number"
                    min="1"
                    max="65535"
                    bind:value={port}
                    placeholder="18799"
                />
            </fieldset>
        </div>

        <fieldset class="fieldset">
            <label class="label" for="mcp-token">{t("mcp_token")}</label>
            <div class="flex gap-2">
                <input
                    id="mcp-token"
                    class="input w-full"
                    type="password"
                    bind:value={token}
                    placeholder={t("mcp_token_placeholder")}
                />
                <button
                    class="btn btn-outline btn-sm"
                    type="button"
                    title={t("mcp_clear_token")}
                    onclick={oncleartoken}
                >
                    {t("mcp_clear_token")}
                </button>
            </div>
            <p class="fieldset-label text-xs">{t("mcp_token_desc")}</p>
        </fieldset>

        <div class="flex flex-wrap items-center gap-2 text-sm">
            {#if status === "running"}
                <span class="badge badge-success">{t("mcp_running")}</span>
                {#if addr}
                    <span class="font-mono text-xs opacity-70"
                        >{addr}/mcp</span
                    >
                {/if}
            {:else if status === "starting"}
                <span class="badge badge-warning gap-1">
                    <span class="loading loading-spinner loading-xs"></span>
                    {t("mcp_starting")}
                </span>
            {:else if status === "error"}
                <span class="badge badge-error">{t("mcp_error")}</span>
                {#if errorMsg}
                    <span class="font-mono text-xs opacity-70"
                        >{errorMsg}</span
                    >
                {/if}
            {:else}
                <span class="badge badge-neutral">{t("mcp_stopped")}</span>
            {/if}
        </div>
    </div>
    <div class="flex justify-end gap-2">
        <button class="btn btn-outline btn-sm" onclick={onsave}
            >{t("mcp_apply")}</button
        >
    </div>
</div>
