<script>
    import InputData from "./InputData.svelte";
    import EmailTemplate from "./EmailTemplate.svelte";
    import ProviderConfig from "./ProviderConfig.svelte";
    import WorkerConfig from "./WorkerConfig.svelte";
    import LogConfig from "./LogConfig.svelte";
    import PreviewModal from "./PreviewModal.svelte";
    import {
        ChevronRight,
        ChevronDown,
        PauseIcon,
        PlayIcon,
        CircleXIcon,
    } from "@lucide/svelte";

    // --- Input Data ---
    let csvText = $state("");
    let csvFile = $state(null);
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
        progress.state === "running" ||
            progress.state === "ready" ||
            progress.state === "paused",
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

    let progressPercent = $derived(
        progress.total > 0
            ? Math.round(
                  ((progress.sent + progress.failed) / progress.total) * 100,
              )
            : 0,
    );

    // --- api ---
    async function apiGet(path) {
        const res = await fetch(path);
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }
    async function apiPost(path, body) {
        const res = await fetch(path, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || res.statusText);
        return data;
    }
    async function apiPostFormData(path, fd) {
        const res = await fetch(path, {
            method: "POST",
            body: fd,
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || res.statusText);
        return data;
    }

    async function loadDefaults() {
        loading = true;
        error = "";
        try {
            const data = await apiGet("/api/campaign/config");
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

            // Restore campaign session on refresh
            const camp = data.campaign;
            if (camp && camp.state !== "idle") {
                // Restore template
                if (camp.template) {
                    subject = camp.template.subject || "";
                    body = camp.template.body || "";
                    toField = camp.template.to || "";
                }
                // Restore config overrides
                if (camp.config) {
                    const c = camp.config;
                    if (c.from) fromEmail = c.from;
                    if (c.provider) provider = c.provider;
                    if (c.smtp?.Host) smtpHost = c.smtp.Host;
                    if (c.smtp?.Port) smtpPort = String(c.smtp.Port);
                    if (c.smtp?.Username) smtpUsername = c.smtp.Username;
                    if (c.smtp?.Password) smtpPassword = c.smtp.Password;
                    if (c.smtp?.TLS !== undefined) smtpTLS = c.smtp.TLS;
                    if (c.ses?.Region) sesRegion = c.ses.Region;
                    if (c.ses?.AccessKeyID) sesAccessKeyId = c.ses.AccessKeyID;
                    if (c.ses?.SecretAccessKey)
                        sesSecretAccessKey = c.ses.SecretAccessKey;
                    if (c.ses?.UseTemplate !== undefined)
                        sesUseTemplate = c.ses.UseTemplate;
                    if (c.ses?.TemplateName)
                        sesTemplateName = c.ses.TemplateName;
                    if (c.ses?.BatchSize) sesBatchSize = c.ses.BatchSize;
                    if (c.worker?.Concurrency)
                        concurrency = c.worker.Concurrency;
                    if (c.worker?.MaxRetries) maxRetries = c.worker.MaxRetries;
                    if (c.smtp_batch_size) smtpBatchSize = c.smtp_batch_size;
                    if (c.log_to_file !== undefined) logToFile = c.log_to_file;
                    if (c.verbose !== undefined) verbose = c.verbose;
                }
                // Restore progress and log
                progress = camp.progress || progress;
                logEvents = camp.events || [];
                logOpen =
                    camp.state === "running" ||
                    camp.state === "paused" ||
                    camp.state === "completed";
            }
        } catch (e) {
            error = "Failed to load config: " + e.message;
        } finally {
            loading = false;
        }
    }

    $effect(() => {
        loadDefaults();
    });

    function buildFormData() {
        const fd = new FormData();
        if (!manualMode && csvFile) {
            fd.append("csv_file", csvFile);
        } else {
            fd.append("csv_text", csvText);
        }
        fd.append("subject", subject);
        fd.append("body", body);
        fd.append("to", toField);
        fd.append("from", fromEmail);
        fd.append("provider", provider);
        fd.append("smtp_host", smtpHost);
        fd.append("smtp_port", smtpPort);
        fd.append("smtp_username", smtpUsername);
        fd.append("smtp_password", smtpPassword);
        fd.append("smtp_tls", String(smtpTLS));
        fd.append("ses_region", sesRegion);
        fd.append("ses_access_key_id", sesAccessKeyId);
        fd.append("ses_secret_access_key", sesSecretAccessKey);
        fd.append("ses_use_template", String(sesUseTemplate));
        fd.append("ses_template_name", sesTemplateName);
        fd.append("ses_batch_size", String(sesBatchSize));
        fd.append("concurrency", String(concurrency));
        fd.append("max_retries", String(maxRetries));
        fd.append("smtp_batch_size", String(smtpBatchSize));
        fd.append("retry_backoff_base", backoffBase);
        fd.append("retry_backoff_max", backoffMax);
        fd.append("log_to_file", String(logToFile));
        fd.append("verbose", String(verbose));
        return fd;
    }

    async function handlePreview() {
        saving = true;
        error = "";
        try {
            const fd = buildFormData();
            fd.append("count", "5");
            const data = await apiPostFormData("/api/campaign/preview", fd);
            previews = data.previews || [];
            previewOpen = true;
        } catch (e) {
            error = "Preview failed: " + e.message;
        } finally {
            saving = false;
        }
    }

    let eventSource;

    function initSSE() {
        eventSource = new EventSource("/api/campaign/events");
        eventSource.onmessage = (e) => {
            try {
                const data = JSON.parse(e.data);
                progress = data.progress;
                logEvents = data.events || [];
            } catch (_) {}
        };
        eventSource.onerror = () => {
            eventSource.close();
            setTimeout(initSSE, 3000);
        };
    }

    $effect(() => {
        initSSE();
    });

    async function handleStart() {
        if (!manualMode && !csvFile) {
            error = "Please provide CSV data first.";
            return;
        }
        if (manualMode && !csvText.trim()) {
            error = "Please provide CSV data first.";
            return;
        }
        saving = true;
        error = "";
        logEvents = [];
        try {
            await apiPostFormData("/api/campaign/start", buildFormData());
            progress.state = "running";
            logOpen = true;
        } catch (e) {
            error = "Start failed: " + e.message;
        } finally {
            saving = false;
        }
    }

    async function handlePause() {
        saving = true;
        try {
            await apiPost("/api/campaign/pause", {});
        } catch (e) {
            error = "Pause failed: " + e.message;
        } finally {
            saving = false;
        }
    }

    async function handleResume() {
        saving = true;
        error = "";
        try {
            await apiPost("/api/campaign/resume", {});
            progress.state = "running";
        } catch (e) {
            error = "Resume failed: " + e.message;
        } finally {
            saving = false;
        }
    }

    async function handleReset() {
        saving = true;
        error = "";
        try {
            await apiPost("/api/campaign/reset", {});
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
            error = "Reset failed: " + e.message;
        } finally {
            saving = false;
        }
    }

    async function handleResetProvider() {
        saving = true;
        try {
            const data = await apiGet("/api/campaign/config");
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
            error = "Failed to reset defaults: " + e.message;
        } finally {
            saving = false;
        }
    }

    async function handleResetWorker() {
        saving = true;
        try {
            const data = await apiGet("/api/campaign/config");
            concurrency = data.worker?.Concurrency || 10;
            maxRetries = data.worker?.MaxRetries || 3;
            backoffBase = data.worker?.RetryBackoffBase?.toString() || "1s";
            backoffMax = data.worker?.RetryBackoffMax?.toString() || "30s";
        } catch (e) {
            error = "Failed to reset defaults: " + e.message;
        } finally {
            saving = false;
        }
    }

    async function handleResetLog() {
        saving = true;
        try {
            const data = await apiGet("/api/campaign/config");
            logToFile = data.log?.campaign?.log_to_file ?? true;
            verbose = data.log?.campaign?.verbose ?? false;
        } catch (e) {
            error = "Failed to reset defaults: " + e.message;
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
                >Dismiss</button
            >
        </div>
    {/if}

    {#if loading}
        <div class="flex items-center justify-center py-20">
            <span class="loading loading-spinner loading-lg"></span>
            <span class="ml-3 text-base-content/70"
                >Loading configuration...</span
            >
        </div>
    {:else}
        <InputData
            bind:csvText
            bind:csvFile
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
            <div role="tablist" class="tabs tabs-lift tabs-xl">
                <button
                    role="tab"
                    class="tab {activeTab === 'provider' ? 'tab-active' : ''}"
                    onclick={() => (activeTab = "provider")}
                    >Email Provider</button
                >
                <button
                    role="tab"
                    class="tab {activeTab === 'worker' ? 'tab-active' : ''}"
                    onclick={() => (activeTab = "worker")}>Worker</button
                >
                <button
                    role="tab"
                    class="tab {activeTab === 'log' ? 'tab-active' : ''}"
                    onclick={() => (activeTab = "log")}>Log</button
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
                    />
                {:else if activeTab === "worker"}
                    <WorkerConfig
                        bind:concurrency
                        bind:maxRetries
                        bind:backoffBase
                        bind:backoffMax
                        disabled={campaignRunning}
                        onreset={handleResetWorker}
                    />
                {:else}
                    <LogConfig
                        bind:logToFile
                        bind:verbose
                        disabled={campaignRunning}
                        onreset={handleResetLog}
                    />
                {/if}
            </div>
        </div>

        <!-- Actions -->
        <div class="card bg-base-100 shadow-sm">
            <div class="card-body">
                <div class="flex flex-wrap gap-3">
                    <button
                        class="btn btn-outline"
                        onclick={handlePreview}
                        disabled={saving ||
                            (!manualMode && !csvFile && !csvText.trim()) ||
                            campaignRunning}
                    >
                        {#if saving}<span
                                class="loading loading-spinner loading-xs"
                            ></span>{/if}
                        Dry-Run Preview
                    </button>
                    {#if progress.state === "running"}
                        <button
                            class="btn btn-warning"
                            onclick={handlePause}
                            disabled={saving}
                        >
                            <PauseIcon class="w-4 h-4"></PauseIcon> Pause
                        </button>
                    {:else if progress.state === "paused"}
                        <button
                            class="btn btn-success"
                            onclick={handleResume}
                            disabled={saving}
                        >
                            <PlayIcon class="w-4 h-4"></PlayIcon> Resume
                        </button>
                    {/if}
                    <button
                        class="btn btn-primary"
                        onclick={handleStart}
                        disabled={saving ||
                            (!manualMode && !csvFile && !csvText.trim()) ||
                            campaignRunning}
                    >
                        {#if campaignRunning}<span
                                class="loading loading-spinner loading-xs"
                            ></span>{/if}
                        <PlayIcon class="w-4 h-4"></PlayIcon> Start Campaign
                    </button>
                    <button
                        class="btn btn-ghost"
                        onclick={handleReset}
                        disabled={saving || progress.state === "running"}
                    >
                        Reset
                    </button>
                </div>
                {#if !csvText.trim()}
                    <p class="text-sm text-base-content/50">
                        Provide CSV data to enable Preview and Start.
                    </p>
                {/if}
            </div>
        </div>

        <!-- Progress -->
        {#if progress.state !== "idle"}
            <div class="card bg-base-100 shadow-sm">
                <div class="card-body">
                    <h2 class="card-title text-lg">
                        Progress
                        {#if progress.state === "running"}
                            <span class="badge badge-info">Running</span>
                        {:else if progress.state === "paused"}
                            <span class="badge badge-warning">Paused</span>
                        {:else if progress.state === "completed"}
                            <span class="badge badge-success">Completed</span>
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
                        {progressPercent}% complete
                    </p>
                    <div class="stats stats-horizontal shadow w-full">
                        <div class="stat">
                            <div class="stat-title">Total</div>
                            <div class="stat-value text-lg">
                                {progress.total}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Sent</div>
                            <div class="stat-value text-lg text-success">
                                {progress.sent}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Failed</div>
                            <div class="stat-value text-lg text-error">
                                {progress.failed}
                            </div>
                        </div>
                        <div class="stat">
                            <div class="stat-title">Pending</div>
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
                        <span>Campaign Log ({logEvents.length})</span>
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
                                    >Waiting for events...</span
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
