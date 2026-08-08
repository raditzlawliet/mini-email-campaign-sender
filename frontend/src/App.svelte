<script>
    import TitleBar from "./lib/components/TitleBar.svelte";
    import WindowResizeHandles from "./lib/components/WindowResizeHandles.svelte";
    import CampaignForm from "./lib/components/CampaignForm.svelte";
</script>

<div class="flex h-screen flex-col overflow-hidden bg-base-200 pt-8">
    <WindowResizeHandles />
    <TitleBar />

    <!-- Scroll is scoped to this container (below the titlebar) so the
         scrollbar never passes through the titlebar. The 6px right/bottom
         insets keep the scrollbar clear of the window resize handles. -->
    <main class="mb-1.5 mr-1.5 min-h-0 flex-1 overflow-y-auto">
        <div class="container mx-auto max-w-4xl px-4 py-4">
            <CampaignForm />
        </div>
    </main>
</div>
