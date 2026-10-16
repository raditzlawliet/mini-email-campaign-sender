<script>
    // A select drawn by the app instead of the OS, so its open list follows
    // the theme. Same look as `.select`; the list is a dropdown `.menu`.
    // Keyboard: Enter/Space/ArrowDown opens, arrows move, Enter/Space picks,
    // Escape or Tab closes.
    import { onMount, tick } from "svelte";

    let {
        id = undefined,
        options = [],
        value = $bindable(),
        disabled = false,
        class: className = "",
    } = $props();

    let open = $state(false);
    let rootRef = $state(null);
    let triggerRef = $state(null);
    let listRef = $state(null);

    const selected = $derived(options.find((o) => o.value === value));

    async function openList() {
        if (disabled) return;
        open = true;
        await tick();
        const current = listRef?.querySelector('[aria-selected="true"]');
        (current ?? listRef?.querySelector('[role="option"]'))?.focus();
    }

    function close(refocus = true) {
        open = false;
        if (refocus) triggerRef?.focus();
    }

    function pick(v) {
        value = v;
        close();
    }

    function onTriggerKey(e) {
        if (["ArrowDown", "ArrowUp", "Enter", " "].includes(e.key)) {
            e.preventDefault();
            openList();
        }
    }

    function onListKey(e) {
        const items = [...listRef.querySelectorAll('[role="option"]')];
        const i = items.indexOf(document.activeElement);
        if (e.key === "ArrowDown") {
            e.preventDefault();
            items[Math.min(i + 1, items.length - 1)]?.focus();
        } else if (e.key === "ArrowUp") {
            e.preventDefault();
            items[Math.max(i - 1, 0)]?.focus();
        } else if (e.key === "Home") {
            e.preventDefault();
            items[0]?.focus();
        } else if (e.key === "End") {
            e.preventDefault();
            items[items.length - 1]?.focus();
        } else if (e.key === "Escape") {
            e.preventDefault();
            close();
        } else if (e.key === "Tab") {
            close(false);
        }
    }

    function handleDocClick(e) {
        if (open && rootRef && !e.composedPath().includes(rootRef)) {
            close(false);
        }
    }

    onMount(() => {
        document.addEventListener("click", handleDocClick);
        return () => document.removeEventListener("click", handleDocClick);
    });
</script>

<div
    class="dropdown block {className}"
    class:dropdown-open={open}
    bind:this={rootRef}
>
    <button
        {id}
        type="button"
        class="select w-full text-start"
        aria-haspopup="listbox"
        aria-expanded={open}
        {disabled}
        bind:this={triggerRef}
        onclick={() => (open ? close() : openList())}
        onkeydown={onTriggerKey}
    >
        <span class="truncate">{selected?.label ?? ""}</span>
    </button>
    {#if open}
        <ul
            class="dropdown-content menu bg-base-100 rounded-box z-10 mt-1 w-full shadow-2xl"
            role="listbox"
            tabindex="-1"
            aria-labelledby={id}
            bind:this={listRef}
            onkeydown={onListKey}
        >
            {#each options as o (o.value)}
                <li>
                    <button
                        type="button"
                        role="option"
                        aria-selected={o.value === value}
                        class:menu-active={o.value === value}
                        onclick={() => pick(o.value)}
                    >
                        {o.label}
                    </button>
                </li>
            {/each}
        </ul>
    {/if}
</div>
