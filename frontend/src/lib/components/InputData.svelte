<script>
    import { t } from "../i18n.svelte.js";

    let {
        csvText = $bindable(""),
        csvFile = $bindable(null),
        csvHeaders = $bindable([]),
        csvCount = $bindable(0),
        manualMode = $bindable(false),
        fileName = $bindable(""),
        disabled = false,
    } = $props();

    let _csvText = $state(csvText);
    $effect(() => {
        _csvText = csvText;
    });

    function flushCSV() {
        csvText = _csvText;
        csvFile = null;
        parseLocal(_csvText);
    }

    function handleFile(e) {
        const file = e.target.files?.[0];
        if (!file) return;
        fileName = file.name;
        csvFile = file;
        csvText = "";
        _csvText = "";
        const reader = new FileReader();
        reader.onload = (ev) => {
            parseLocal(ev.target.result);
        };
        reader.readAsText(file);
    }

    function toggleManual() {
        manualMode = !manualMode;
        csvFile = null;
        if (manualMode) {
            fileName = "";
            _csvText = "";
            csvText = "";
            parseLocal("");
            const input = document.getElementById("csv-file-input");
            if (input) input.value = "";
        } else {
            _csvText = "";
            csvText = "";
            parseLocal("");
        }
    }

    function parseLocal(text) {
        if (!text.trim()) {
            csvHeaders = [];
            csvCount = 0;
            return;
        }
        const lines = text.trim().split("\n");
        if (lines.length < 2) {
            csvHeaders = [];
            csvCount = 0;
            return;
        }
        csvHeaders = lines[0].split(",").map((h) => h.trim());
        csvCount = lines.length - 1;
    }

    let toast = $state("");
    let toastTimer;

    async function copyHeader(h) {
        const key = `{${h}}`;
        await navigator.clipboard.writeText(key);
        toast = key;
        clearTimeout(toastTimer);
        toastTimer = setTimeout(() => (toast = ""), 2000);
    }
</script>

<div class="card bg-base-100 shadow-sm">
    <div class="card-body">
        <div class="flex items-center justify-between">
            <h2 class="card-title text-lg">{t("input_data")}</h2>
            <label class="label cursor-pointer gap-2">
                <span class="label-text">{t("manual_input")}</span>
                <input
                    type="checkbox"
                    class="toggle"
                    checked={manualMode}
                    onchange={toggleManual}
                    {disabled}
                />
            </label>
        </div>

        {#if !manualMode}
            <input
                id="csv-file-input"
                type="file"
                accept=".csv"
                class="file-input file-input-bordered w-full"
                onchange={handleFile}
                {disabled}
            />
        {:else}
            <textarea
                class="textarea textarea-bordered font-mono text-sm w-full"
                rows="5"
                placeholder="name,email&#10;Alice,alice@example.com&#10;Bob,bob@example.com"
                bind:value={_csvText}
                onblur={flushCSV}
                {disabled}></textarea>
        {/if}

        <div class="text-xs text-base-content/50 leading-relaxed">
            {t("csv_format")} <strong>{t("csv")}</strong>
            {t("with_header_row_requires")}
            <code class="badge badge-xs badge-ghost">{t("email_col")}</code>
            {t("all_columns_become")}
            {"{"}<code class="badge badge-xs badge-ghost">name</code>{"}"}.
        </div>

        {#if !manualMode}
            {#if fileName}
                <p class="text-sm text-base-content/60">
                    {fileName} — {csvCount}
                    {t("recipients")}
                </p>
            {/if}
        {:else}
            {#if csvCount > 0}
                <p class="text-sm text-base-content/60">
                    {csvCount}
                    {t("recipients")}
                </p>
            {/if}
        {/if}

        {#if csvHeaders.length > 0}
            <div class="flex flex-wrap items-center gap-2 text-sm">
                <span>{t("available")} </span>
                {#each csvHeaders as h}
                    <button
                        class="badge badge-outline cursor-pointer hover:badge-primary transition-colors"
                        onclick={() => copyHeader(h)}
                        title="Click to copy {'{' + h + '}'}"
                    >
                        {h}
                    </button>
                {/each}
            </div>
        {/if}

        {#if toast}
            <div class="toast toast-end toast-bottom">
                <div class="alert alert-success text-sm py-2">
                    <span>{toast} {t("copied_to_clipboard")}</span>
                </div>
            </div>
        {/if}
    </div>
</div>
