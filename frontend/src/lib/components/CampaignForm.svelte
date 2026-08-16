<script>
    import InputData from "./InputData.svelte";
    import EmailTemplate from "./EmailTemplate.svelte";
    import ProviderConfig from "./ProviderConfig.svelte";
    import WorkerConfig from "./WorkerConfig.svelte";
    import LogConfig from "./LogConfig.svelte";
    import PreviewModal from "./PreviewModal.svelte";
    import ConfirmModal from "./ConfirmModal.svelte";
    import {
        ChevronRight,
        ChevronDown,
        PauseIcon,
        PlayIcon,
        CircleXIcon,
    } from "@lucide/svelte";
    import { t } from "../i18n.svelte.js";
    import {
        GetCampaignConfig,
        GetVersion,
        Preview,
        StartCampaign,
        PauseCampaign,
        ResumeCampaign,
        ResetCampaign,
        SaveConfig,
    } from "../wailsjs/go/app/App";
    import { EventsOn } from "../wailsjs/runtime/runtime";

    // --- Input Data ---
    let csvText = $state("");
    let csvFilePath = $state("");
    let csvHeaders = $state([]);
    let csvCount = $state(0);
    let manualMode = $state(false);
    let fileName = $state("");

    // --- Template ---
    let subject = $state("");
    let body = $state("");
    let toField = $state("{name} <{email}>");

    // --- Provider ---
    let provider = $state("smtp");
    let fromEmail = $state("");
    let smtpHost = $state("");
    let smtpPort = $state("");
    let smtpUsername = $state("");
    let smtpPassword = $state("");
    let smtpTLS = $state(false);
    let smtpBatchSize = $state(50);
    let sesRegion = $state("");
    let sesAccessKeyId = $state("");
    let sesSecretAccessKey = $state("");
    let sesUseTemplate = $state(false);
    let sesTemplateName = $state("");
    let sesBatchSize = $state(50);

    // --- Worker ---
    let concurrency = $state(10);
    let maxRetries = $state(3);
    let backoffBase = $state("1s");
    let backoffMax = $state("30s");

    // --- Log ---
    let logToFile = $state(true);
    let verbose = $state(false);

    // --- UI state ---
    let activeTab = $state("provider");
    let previews = $state([]);
    let previewOpen = $state(false);
    let progress = $state({
        total: 0,
        sent: 0,
        failed: 0,
        pending: 0,
        state: "idle",
    });
    let campaignRunning = $derived(
        progress.state === "running" || progress.state === "paused",
    );
    let loading = $state(true);
    let error = $state("");
    let saving = $state(false);

    // --- Log ---
    let logEvents = $state([]);
    let logOpen = $state(false);
    let logRef = $state(null);

    $effect(() => {
        if (logRef && logEvents.length) {
            const el = logRef;
            const atBottom =
                el.scrollHeight - el.scrollTop - el.clientHeight < 50;
            if (atBottom || logEvents.length <= 1) {
                el.scrollTop = el.scrollHeight;
            }
        }
    });

    // --- Confirm dialog ---
    let confirmOpen = $state(false);
    let confirmTitle = $state("");
    let confirmAction = $state(() => {});

    function showConfirm(title, action) {
        confirmTitle = title;
        confirmAction = action;
        confirmOpen = true;
    }

    // --- Save handlers ---
    async function handleSaveProvider() {
        const payload = {
            email: {
                provider,
                from: fromEmail,
                smtp: {
                    host: smtpHost,
                    port: parseInt(smtpPort) || 0,
                    username: smtpUsername,
                    password: smtpPassword,
                    tls: smtpTLS,
                    batch_size: parseInt(smtpBatchSize) || 50,
                },
                ses: {
                    region: sesRegion,
                    access_key_id: sesAccessKeyId,
                    secret_access_key: sesSecretAccessKey,
                    use_template: sesUseTemplate,
                    template_name: sesTemplateName,
                    batch_size: parseInt(sesBatchSize) || 50,
                },
            },
        };
        await doSave(payload);
    }

    async function handleSaveWorker() {
        const payload = {
            worker: {
                concurrency: parseInt(concurrency) || 10,
                max_retries: parseInt(maxRetries) || 3,
                retry_backoff_base: backoffBase || "1s",
                retry_backoff_max: backoffMax || "30s",
            },
        };
        await doSave(payload);
    }

    async function handleSaveLog() {
        const payload = {
            log: {
                campaign: {
                    log_to_file: logToFile,
                    verbose,
                },
            },
        };
        await doSave(payload);
    }

    async function doSave(payload) {
        saving = true;
        error = "";
        try {
            await SaveConfig(JSON.stringify(payload));
        } catch (e) {
            error = t("save_config_failed") + " " + e.message;
        } finally {
            saving = false;
        }
    }

    let progressPercent = $derived(
        progress.total > 0
            ? Math.round(
                  ((progress.sent + progress.failed) / progress.total) * 100,
              )
            : 0,
    );

    // --- api (Wails bindings) ---
    // Bound methods return Promises; Go errors reject with an Error.
    function bindingError(e) {
        const msg = e && e.message ? e.message : String(e);
        return msg.replace(/^Error: /, "");
    }

    async function loadDefaults() {
        loading = true;
        error = "";
        try {
            const data = await GetCampaignConfig();
            // Default config
            fromEmail = data.email.from || "";
            provider = data.email.provider || "smtp";
            smtpHost = data.email.smtp?.Host || "";
            smtpPort = data.email.smtp?.Port?.toString() || "";
            smtpUsername = data.email.smtp?.Username || "";
            smtpPassword = data.email.smtp?.Password || "";
            smtpTLS = data.email.smtp?.TLS || false;
            smtpBatchSize = data.email.smtp?.BatchSize || 50;
            sesRegion = data.email.ses?.Region || "";
            sesAccessKeyId = data.email.ses?.AccessKeyID || "";
            sesSecretAccessKey = data.email.ses?.SecretAccessKey || "";
            sesUseTemplate = data.email.ses?.UseTemplate || false;
            sesTemplateName = data.email.ses?.TemplateName || "";
            sesBatchSize = data.email.ses?.BatchSize || 50;
            concurrency = data.worker?.Concurrency || 10;
            maxRetries = data.worker?.MaxRetries || 3;
            backoffBase = data.worker?.RetryBackoffBase?.toString() || "1s";
            backoffMax = data.worker?.RetryBackoffMax?.toString() || "30s";
            logToFile = data.log?.campaign?.log_to_file ?? true;
            verbose = data.log?.campaign?.verbose ?? false;

            // Restore campaign session on refresh (and on MCP changes)
            syncFromStore(data.campaign);
            const campRev = data.campaign?.revision ?? -1;
            lastSeenRevision = campRev;
            lastAppliedRevision = campRev;
        } catch (e) {
            error = t("load_config_failed") + " " + e.message;
        } finally {
            loading = false;
        }
    }

    // Applies the backend campaign snapshot (template, config, CSV) to the
    // form. Used on session restore and whenever MCP stages a campaign.
    let lastSeenRevision = $state(-1);
    let lastAppliedRevision = $state(-1);
    let lastAppliedState = $state(null);
    let csvTooLarge = $state(false);
    const MAX_CSV_TEXT = 100000; // chars; larger CSV is too heavy for a textarea

    // syncFromStore mirrors backend campaign state into the form. Idle with a
    // prior staged state means the store was cleared (MCP clear / reset), so
    // the form resets too. Config-only updates (e.g. smtp_batch_size via MCP)
    // still apply while idle.
    function syncFromStore(camp) {
        if (!camp) return;
        if (camp.state === "idle") {
            if (!isConfigEmpty(camp.config)) {
                applyConfig(camp.config);
            }
            if (lastAppliedState && lastAppliedState !== "idle") {
                subject = "";
                body = "";
                toField = "";
                csvText = "";
                csvCount = 0;
                manualMode = false;
                csvTooLarge = false;
                logEvents = [];
            }
            lastAppliedState = "idle";
            return;
        }
        lastAppliedState = camp.state;
        applyCampaign(camp);
    }

    function isConfigEmpty(c) {
        return !c || (
            !c.from &&
            !c.provider &&
            !c.smtp?.Host &&
            !c.smtp?.Port &&
            !c.smtp?.Username &&
            !c.smtp?.Password &&
            !c.smtp?.BatchSize &&
            !c.smtp?.TLS &&
            !c.ses?.Region &&
            !c.ses?.AccessKeyID &&
            !c.ses?.SecretAccessKey &&
            !c.ses?.TemplateName &&
            !c.ses?.BatchSize &&
            !c.ses?.UseTemplate &&
            !c.worker?.Concurrency &&
            !c.worker?.MaxRetries &&
            !c.smtp_batch_size &&
            !c.log_to_file &&
            !c.verbose
        );
    }

    function applyConfig(c) {
        if (!c) return;
        if (c.from) fromEmail = c.from;
        if (c.provider) provider = c.provider;
        if (c.smtp?.Host) smtpHost = c.smtp.Host;
        if (c.smtp?.Port) smtpPort = String(c.smtp.Port);
        if (c.smtp?.Username) smtpUsername = c.smtp.Username;
        if (c.smtp?.Password) smtpPassword = c.smtp.Password;
        if (c.smtp?.TLS !== undefined) smtpTLS = c.smtp.TLS;
        if (c.smtp?.BatchSize) smtpBatchSize = c.smtp.BatchSize;
        if (c.ses?.Region) sesRegion = c.ses.Region;
        if (c.ses?.AccessKeyID) sesAccessKeyId = c.ses.AccessKeyID;
        if (c.ses?.SecretAccessKey)
            sesSecretAccessKey = c.ses.SecretAccessKey;
        if (c.ses?.UseTemplate !== undefined)
            sesUseTemplate = c.ses.UseTemplate;
        if (c.ses?.TemplateName) sesTemplateName = c.ses.TemplateName;
        if (c.ses?.BatchSize) sesBatchSize = c.ses.BatchSize;
        if (c.worker?.Concurrency) concurrency = c.worker.Concurrency;
        if (c.worker?.MaxRetries) maxRetries = c.worker.MaxRetries;
        if (c.smtp_batch_size) smtpBatchSize = c.smtp_batch_size;
        if (c.log_to_file !== undefined) logToFile = c.log_to_file;
        if (c.verbose !== undefined) verbose = c.verbose;
    }

    function applyCampaign(camp) {
        if (!camp) return;
        // Template
        if (camp.template) {
            subject = camp.template.subject || "";
            body = camp.template.body || "";
            toField = camp.template.to || "";
        }
        // Config overrides
        applyConfig(camp.config);
        // CSV prepared via MCP shows in manual mode
        if (camp.csv_text) {
            manualMode = true;
            csvCount = camp.progress?.total || 0;
            if (camp.csv_text.length <= MAX_CSV_TEXT) {
                csvText = camp.csv_text;
                csvTooLarge = false;
            } else {
                // Too large for the editor: keep the staged data in the store,
                // but never submit a stale csvText from a previous sync.
                csvText = "";
                csvTooLarge = true;
            }
        }
        // Progress + log
        progress = camp.progress || progress;
        if (camp.events) logEvents = camp.events;
        logOpen =
            camp.state === "running" ||
            camp.state === "paused" ||
            camp.state === "completed";
    }

    $effect(() => {
        loadDefaults();
    });

    function buildInput() {
        return {
            CSVText: manualMode ? csvText : "",
            CSVFilePath: !manualMode ? csvFilePath : "",
            Subject: subject,
            Body: body,
            To: toField,
            From: fromEmail,
            Provider: provider,
            SMTPHost: smtpHost,
            SMTPPort: parseInt(smtpPort) || 0,
            SMTPUsername: smtpUsername,
            SMTPPassword: smtpPassword,
            SMTPTLS: smtpTLS,
            SMTPBatchSize: parseInt(smtpBatchSize) || 50,
            SESRegion: sesRegion,
            SESAccessKeyID: sesAccessKeyId,
            SESSecretKey: sesSecretAccessKey,
            SESUseTemplate: sesUseTemplate,
            SESTemplateName: sesTemplateName,
            SESBatchSize: parseInt(sesBatchSize) || 50,
            Concurrency: parseInt(concurrency) || 10,
            MaxRetries: parseInt(maxRetries) || 3,
            BackoffBase: backoffBase,
            BackoffMax: backoffMax,
            LogToFile: logToFile,
            Verbose: verbose,
            Count: 0,
        };
    }

    async function handlePreview() {
        saving = true;
        error = "";
        try {
            const input = buildInput();
            input.Count = 5;
            previews = await Preview(input);
            previewOpen = true;
        } catch (e) {
            error = t("preview_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    $effect(() => {
        // Wails Events replace the old SSE stream; never disconnected.
        EventsOn("campaign:progress", (data) => {
            if (data && data.progress) progress = data.progress;
            if (data && data.events) logEvents = data.events;
            // Campaign staged via MCP (or cleared) - mirror it into the form.
            if (
                data &&
                data.revision !== undefined &&
                data.revision !== lastSeenRevision
            ) {
                const target = data.revision;
                const prev = lastSeenRevision;
                lastSeenRevision = target;
                GetCampaignConfig()
                    .then((cfg) => {
                        const campRev = cfg.campaign?.revision ?? target;
                        // Ignore stale snapshots arriving out of order.
                        if (campRev < lastAppliedRevision) return;
                        syncFromStore(cfg.campaign);
                        lastAppliedRevision = campRev;
                    })
                    .catch(() => {
                        // Keep the seen mark at the previous value so the next
                        // tick retries instead of suppressing the sync.
                        lastSeenRevision = prev;
                    });
            }
        });
    });

    async function handleStart() {
        if (csvTooLarge) {
            error = t("csv_too_large");
            return;
        }
        if (!manualMode && !csvFilePath) {
            error = t("please_provide_csv");
            return;
        }
        if (manualMode && !csvText.trim()) {
            error = t("please_provide_csv");
            return;
        }
        saving = true;
        error = "";
        logEvents = [];
        try {
            await StartCampaign(buildInput());
            progress.state = "running";
            logOpen = true;
        } catch (e) {
            error = t("start_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    async function handlePause() {
        saving = true;
        try {
            await PauseCampaign();
        } catch (e) {
            error = t("pause_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    async function handleResume() {
        saving = true;
        error = "";
        try {
            await ResumeCampaign();
            progress.state = "running";
        } catch (e) {
            error = t("resume_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    async function handleReset() {
        saving = true;
        error = "";
        try {
            await ResetCampaign();
            progress = {
                total: 0,
                sent: 0,
                failed: 0,
                pending: 0,
                state: "idle",
            };
            logEvents = [];
            logOpen = false;
        } catch (e) {
            error = t("reset_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    async function handleResetProvider() {
        saving = true;
        try {
            const data = await GetCampaignConfig();
            fromEmail = data.email.from || "";
            provider = data.email.provider || "smtp";
            smtpHost = data.email.smtp?.Host || "";
            smtpPort = data.email.smtp?.Port?.toString() || "";
            smtpUsername = data.email.smtp?.Username || "";
            smtpPassword = data.email.smtp?.Password || "";
            smtpTLS = data.email.smtp?.TLS || false;
            smtpBatchSize = data.email.smtp?.BatchSize || 50;
            sesRegion = data.email.ses?.Region || "";
            sesAccessKeyId = data.email.ses?.AccessKeyID || "";
            sesSecretAccessKey = data.email.ses?.SecretAccessKey || "";
            sesUseTemplate = data.email.ses?.UseTemplate || false;
            sesTemplateName = data.email.ses?.TemplateName || "";
            sesBatchSize = data.email.ses?.BatchSize || 50;
        } catch (e) {
            error = t("reset_defaults_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    async function handleResetWorker() {
        saving = true;
        try {
            const data = await GetCampaignConfig();
            concurrency = data.worker?.Concurrency || 10;
            maxRetries = data.worker?.MaxRetries || 3;
            backoffBase = data.worker?.RetryBackoffBase?.toString() || "1s";
            backoffMax = data.worker?.RetryBackoffMax?.toString() || "30s";
        } catch (e) {
            error = t("reset_defaults_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }

    async function handleResetLog() {
        saving = true;
        try {
            const data = await GetCampaignConfig();
            logToFile = data.log?.campaign?.log_to_file ?? true;
            verbose = data.log?.campaign?.verbose ?? false;
        } catch (e) {
            error = t("reset_defaults_failed") + " " + bindingError(e);
        } finally {
            saving = false;
        }
    }
</script>

<div class="space-y-6">
    {#if error}
        <div role="alert" class="alert alert-error alert-soft">
            <CircleXIcon class="w-4 h-4"></CircleXIcon>
            <span>{error}</span>
            <button class="btn btn-ghost btn-sm" onclick={() => (error = "")}
                >{t("dismiss")}</button
            >
        </div>
    {/if}

    {#if loading}
        <div class="flex items-center justify-center py-20">
            <span class="loading loading-spinner loading-lg"></span>
            <span class="ml-3 text-base-content/70">{t("loading_config")}</span>
        </div>
    {:else}
        <InputData
            bind:csvText
            bind:csvFilePath
            bind:csvHeaders
            bind:csvCount
            bind:manualMode
            bind:fileName
            disabled={campaignRunning}
        />

        <EmailTemplate
            bind:toField
            bind:subject
            bind:body
            useTemplate={sesUseTemplate}
            disabled={campaignRunning}
        />

        <!-- Config Tabs -->
        <div class="card">
            <div role="tablist" class="tabs tabs-lift">
                <button
                    role="tab"
                    class="tab {activeTab === 'provider'
                        ? 'tab-active font-medium'
                        : ''}"
                    onclick={() => (activeTab = "provider")}
                    >{t("email_provider_tab")}</button
                >
                <button
                    role="tab"
                    class="tab {activeTab === 'worker'
                        ? 'tab-active font-medium'
                        : ''}"
                    onclick={() => (activeTab = "worker")}
                    >{t("worker_tab")}</button
                >
                <button
                    role="tab"
                    class="tab {activeTab === 'log'
                        ? 'tab-active font-medium'
                        : ''}"
                    onclick={() => (activeTab = "log")}>{t("log_tab")}</button
                >
            </div>

            <div class="card-body bg-base-100 shadow-sm">
                {#if activeTab === "provider"}
                    <ProviderConfig
                        bind:provider
                        bind:fromEmail
                        bind:smtpHost
                        bind:smtpPort
                        bind:smtpUsername
                        bind:smtpPassword
                        bind:smtpTLS
                        bind:smtpBatchSize
                        bind:sesRegion
                        bind:sesAccessKeyId
                        bind:sesSecretAccessKey
                        bind:sesUseTemplate
                        bind:sesTemplateName
                        bind:sesBatchSize
                        disabled={campaignRunning}
                        onreset={handleResetProvider}
                        onsave={() =>
                            showConfirm(
                                t("save_provider_confirm"),
                                handleSaveProvider,
                            )}
                    />
                {:else if activeTab === "worker"}
                    <WorkerConfig
                        bind:concurrency
                        bind:maxRetries
                        bind:backoffBase
                        bind:backoffMax
                        disabled={campaignRunning}
                        onreset={handleResetWorker}
                        onsave={() =>
                            showConfirm(
                                t("save_worker_confirm"),
                                handleSaveWorker,
                            )}
                    />
                {:else}
                    <LogConfig
                        bind:logToFile
                        bind:verbose
                        disabled={campaignRunning}
                        onreset={handleResetLog}
                        onsave={() =>
                            showConfirm(t("save_log_confirm"), handleSaveLog)}
                    />
                {/if}
            </div>
        </div>

        <!-- Actions -->
        {#if csvTooLarge}
            <div class="alert alert-warning text-sm">
                {t("csv_too_large")}
            </div>
        {/if}
        <div class="card bg-base-100 shadow-sm">
            <div class="card-body">
                <div class="flex flex-wrap gap-3">
                    <button
                        class="btn btn-outline"
                        onclick={handlePreview}
                        disabled={saving ||
                            (!manualMode && !csvFilePath && !csvText.trim()) ||
                            campaignRunning}
                    >
                        {#if saving}<span
                                class="loading loading-spinner loading-xs"
                            ></span>{/if}
                        {t("dry_run_preview")}
                    </button>
                    {#if progress.state === "running"}
                        <button
                            class="btn btn-warning"
                            onclick={handlePause}
                            disabled={saving}
                        >
                            <PauseIcon class="w-4 h-4"></PauseIcon>
                            {t("pause")}
                        </button>
                    {:else if progress.state === "paused"}
                        <button
                            class="btn btn-success"
                            onclick={handleResume}
                            disabled={saving}
                        >
                            <PlayIcon class="w-4 h-4"></PlayIcon>
                            {t("resume")}
                        </button>
                    {/if}
                    <button
                        class="btn btn-primary"
                        onclick={handleStart}
                        disabled={saving ||
                            (!manualMode && !csvFilePath && !csvText.trim()) ||
                            campaignRunning}
                    >
                        {#if campaignRunning}<span
                                class="loading loading-spinner loading-xs"
                            ></span>{/if}
                        <PlayIcon class="w-4 h-4"></PlayIcon>
                        {t("start_campaign")}
                    </button>
                    <button
                        class="btn btn-ghost"
                        onclick={handleReset}
                        disabled={saving || progress.state === "running"}
                    >
                        {t("reset")}
                    </button>
                </div>
                {#if !csvText.trim()}
                    <p class="text-sm text-base-content/50">
                        {t("provide_csv")}
                    </p>
                {/if}
            </div>
        </div>

        <!-- Progress -->
        {#if progress.state !== "idle"}
            <div class="card bg-base-100 shadow-sm">
                <div class="card-body">
                    <h2 class="card-title text-lg">
                        {t("progress")}
                        {#if progress.state === "running"}
                            <span class="badge badge-info">{t("running")}</span>
                        {:else if progress.state === "paused"}
                            <span class="badge badge-warning"
                                >{t("paused")}</span
                            >
                        {:else if progress.state === "completed"}
                            <span class="badge badge-success"
                                >{t("completed")}</span
                            >
                        {:else}
                            <span class="badge">{progress.state}</span>
                        {/if}
                    </h2>
                    <progress
                        class="progress progress-primary w-full"
                        value={progressPercent}
                        max="100"
                    ></progress>
                    <p class="text-sm text-base-content/60">
                        {progressPercent}% {t("complete_label")}
                    </p>
                    <div class="stats stats-horizontal shadow w-full">
                        <div class="stat">
                            <div class="stat-title">{t("total")}</div>
                            <div class="stat-value text-lg">
                                {progress.total}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">{t("sent")}</div>
                            <div class="stat-value text-lg text-success">
                                {progress.sent}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">{t("failed")}</div>
                            <div class="stat-value text-lg text-error">
                                {progress.failed}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">{t("pending")}</div>
                            <div class="stat-value text-lg text-warning">
                                {progress.pending}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Log -->
            <div class="card bg-base-100 shadow-sm mt-4">
                <div class="card-body p-3">
                    <button
                        class="flex items-center justify-between w-full text-sm font-semibold"
                        onclick={() => (logOpen = !logOpen)}
                    >
                        <span>{t("campaign_log")} ({logEvents.length})</span>
                        <span class="text-xs">
                            {#if logOpen}
                                <ChevronDown class="w-4 h-4" />
                            {:else}
                                <ChevronRight class="w-4 h-4" />
                            {/if}
                        </span>
                    </button>
                    {#if logOpen}
                        <div
                            class="bg-base-300 rounded-box p-2 max-h-48 overflow-y-auto font-mono text-xs space-y-0.5"
                            bind:this={logRef}
                        >
                            {#if logEvents.length === 0}
                                <span class="text-base-content/40"
                                    >{t("waiting_events")}</span
                                >
                            {:else}
                                {#each logEvents as ev}
                                    <div class="flex gap-2">
                                        <span
                                            class="text-base-content/40 shrink-0"
                                            >{ev.time?.slice(11, 19) ||
                                                ""}</span
                                        >
                                        <span
                                            class={ev.level === "error"
                                                ? "text-error"
                                                : ev.level === "warn"
                                                  ? "text-warning"
                                                  : "text-base-content"}
                                            >{ev.message}</span
                                        >
                                    </div>
                                {/each}
                            {/if}
                        </div>
                    {/if}
                </div>
            </div>
        {/if}
    {/if}
</div>

<PreviewModal
    open={previewOpen}
    {previews}
    onclose={() => (previewOpen = false)}
/>

<ConfirmModal
    open={confirmOpen}
    title={confirmTitle}
    onconfirm={() => {
        confirmAction();
        confirmOpen = false;
    }}
    oncancel={() => (confirmOpen = false)}
/>
