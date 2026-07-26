<script>
    import InputData from "./InputData.svelte";
    import EmailTemplate from "./EmailTemplate.svelte";
    import ProviderConfig from "./ProviderConfig.svelte";
    import WorkerConfig from "./WorkerConfig.svelte";
    import PreviewModal from "./PreviewModal.svelte";
    import {
        ChevronRight,
        ChevronDown,
        PauseIcon,
        PlayIcon,
    } from "@lucide/svelte";

    // --- Input Data ---
    let csvText = $state("");
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
    let sesRegion = $state("");
    let sesAccessKeyId = $state("");
    let sesSecretAccessKey = $state("");

    // --- Worker ---
    let concurrency = $state(10);
    let maxRetries = $state(3);
    let backoffBase = $state("1s");
    let backoffMax = $state("30s");

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
            sesRegion = data.email.ses?.Region || "";
            sesAccessKeyId = data.email.ses?.AccessKeyID || "";
            sesSecretAccessKey = data.email.ses?.SecretAccessKey || "";
            concurrency = data.worker?.Concurrency || 10;
            maxRetries = data.worker?.MaxRetries || 3;
            backoffBase = data.worker?.RetryBackoffBase?.toString() || "1s";
            backoffMax = data.worker?.RetryBackoffMax?.toString() || "30s";

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
                    if (c.ses?.SecretAccessKey) sesSecretAccessKey = c.ses.SecretAccessKey;
                    if (c.worker?.Concurrency) concurrency = c.worker.Concurrency;
                    if (c.worker?.MaxRetries) maxRetries = c.worker.MaxRetries;
                }
                // Restore progress and log
                progress = camp.progress || progress;
                logEvents = camp.events || [];
                logOpen = camp.state === "running" || camp.state === "paused" || camp.state === "completed";
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

    function buildPayload() {
        return {
            csv: csvText,
            subject,
            body,
            to: toField,
            from: fromEmail,
            provider,
            smtp: {
                Host: smtpHost,
                Port: parseInt(smtpPort) || 0,
                Username: smtpUsername,
                Password: smtpPassword,
                TLS: smtpTLS,
            },
            ses: {
                Region: sesRegion,
                AccessKeyID: sesAccessKeyId,
                SecretAccessKey: sesSecretAccessKey,
            },
            concurrency,
            max_retries: maxRetries,
            retry_backoff_base: backoffBase,
            retry_backoff_max: backoffMax,
        };
    }

    async function handlePreview() {
        saving = true;
        error = "";
        try {
            const payload = buildPayload();
            payload.count = 5;
            const data = await apiPost("/api/campaign/preview", payload);
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

    $effect(() => { initSSE(); });

    async function handleStart() {
        if (!csvText.trim()) {
            error = "Please provide CSV data first.";
            return;
        }
        saving = true;
        error = "";
        logEvents = [];
        try {
            await apiPost("/api/campaign/start", buildPayload());
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
            sesRegion = data.email.ses?.Region || "";
            sesAccessKeyId = data.email.ses?.AccessKeyID || "";
            sesSecretAccessKey = data.email.ses?.SecretAccessKey || "";
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
</script>

<div class="space-y-6">
    {#if error}
        <div role="alert" class="alert alert-error">
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-6 w-6 shrink-0 stroke-current"
                fill="none"
                viewBox="0 0 24 24"
                ><path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
                /></svg
            >
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
            disabled={campaignRunning}
        />

        <!-- Config Tabs -->
        <div class="card">
            <div role="tablist" class="tabs tabs-lift">
                <button
                    role="tab"
                    class="tab {activeTab === 'provider' ? 'tab-active' : ''}"
                    onclick={() => (activeTab = "provider")}
                    >Email Provider</button
                >
                <button
                    role="tab"
                    class="tab {activeTab === 'worker' ? 'tab-active' : ''}"
                    onclick={() => (activeTab = "worker")}>Worker Config</button
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
                        bind:sesRegion
                        bind:sesAccessKeyId
                        bind:sesSecretAccessKey
                        disabled={campaignRunning}
                        onreset={handleResetProvider}
                    />
                {:else}
                    <WorkerConfig
                        bind:concurrency
                        bind:maxRetries
                        bind:backoffBase
                        bind:backoffMax
                        disabled={campaignRunning}
                        onreset={handleResetWorker}
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
                        disabled={saving || !csvText.trim() || campaignRunning}
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
                        disabled={saving || !csvText.trim() || campaignRunning}
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
