<script>
    let {
        csvText = $bindable(""),
        csvHeaders = $bindable([]),
        csvCount = $bindable(0),
        manualMode = $bindable(false),
        fileName = $bindable(""),
    } = $props();

    let _csvText = $state(csvText);
    $effect(() => { _csvText = csvText; });

    function flushCSV() {
        csvText = _csvText;
        parseLocal(_csvText);
    }

    function handleFile(e) {
        const file = e.target.files?.[0];
        if (!file) return;
        fileName = file.name;
        const reader = new FileReader();
        reader.onload = (ev) => {
            _csvText = ev.target.result;
            csvText = _csvText;
            parseLocal(_csvText);
        };
        reader.readAsText(file);
    }

    function toggleManual() {
        manualMode = !manualMode;
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
            <h2 class="card-title text-lg">Input Data</h2>
            <label class="label cursor-pointer gap-2">
                <span class="label-text text-sm">Manual input</span>
                <input type="checkbox" class="toggle toggle-sm" checked={manualMode} onchange={toggleManual} />
            </label>
        </div>

        {#if !manualMode}
            <input id="csv-file-input" type="file" accept=".csv"
                class="file-input file-input-bordered w-full" onchange={handleFile} />
        {:else}
            <textarea class="textarea textarea-bordered font-mono text-sm w-full" rows="5"
                placeholder="name,email&#10;Alice,alice@example.com&#10;Bob,bob@example.com"
                bind:value={_csvText} onblur={flushCSV}></textarea>
        {/if}

        <div class="text-xs text-base-content/50 leading-relaxed">
            Format: <strong>CSV</strong> with header row. Requires an
            <code class="badge badge-xs badge-ghost">email</code>
            column. All columns become template placeholders, e.g. {"{"}<code class="badge badge-xs badge-ghost">name</code>{"}"}.
        </div>

        {#if !manualMode}
            {#if fileName}
                <p class="text-sm text-base-content/60">{fileName} — {csvCount} recipient(s)</p>
            {/if}
        {:else}
            {#if csvCount > 0}
                <p class="text-sm text-base-content/60">{csvCount} recipient(s)</p>
            {/if}
        {/if}

        {#if csvHeaders.length > 0}
            <div class="flex flex-wrap items-center gap-2 text-sm">
                <span>Available: </span>
                {#each csvHeaders as h}
                    <button
                        class="badge badge-outline cursor-pointer hover:badge-primary transition-colors"
                        onclick={() => copyHeader(h)}
                        title="Click to copy {'{'+h+'}'}">
                        {h}
                    </button>
                {/each}
            </div>
        {/if}

        {#if toast}
            <div class="toast toast-end toast-bottom">
                <div class="alert alert-success text-sm py-2">
                    <span>{toast} copied to clipboard</span>
                </div>
            </div>
        {/if}
    </div>
</div>
