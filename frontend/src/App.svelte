<script>
    import CampaignForm from "./lib/components/CampaignForm.svelte";
</script>

<div class="min-h-screen bg-base-200">
    <div class="navbar bg-base-100 shadow-sm">
        <div class="navbar-start"></div>
        <div class="navbar-center">
            <span class="font-bold">Mini Email Campaign Sender</span>
        </div>
        <div class="navbar-end"></div>
    </div>

    <main class="container mx-auto max-w-4xl px-4 py-8">
        <CampaignForm />
    </main>
</div>
