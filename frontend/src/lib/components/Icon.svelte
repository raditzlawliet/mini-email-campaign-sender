<script>
    // An icon that follows the theme: the smooth Lucide icon (or a children
    // snippet) in the stock DaisyUI themes, and the pixel icon from
    // lib/icons.js in the Traveler Gilded themes. Both are rendered; CSS in
    // app.css and themes/traveler-gilded.css shows the one that fits.
    // Decorative only (aria-hidden): label the button that holds it.
    import { iconRects } from "../icons.js";

    let {
        pixel,
        smooth: Smooth = null,
        size = 24, // pixel size: 12, 24 or 48
        class: className = "size-4",
        pixelClass = "",
        children,
    } = $props();

    const rects = $derived(iconRects(pixel));
</script>

{#if Smooth}
    <Smooth class="app-icon-smooth {className}" aria-hidden="true" />
{:else if children}
    <span class="app-icon-smooth contents">{@render children()}</span>
{/if}
<svg
    class="app-icon-pixel {pixelClass}"
    viewBox="0 0 12 12"
    width={size}
    height={size}
    fill="currentColor"
    shape-rendering="crispEdges"
    aria-hidden="true"
    focusable="false"
>
    {#each rects as r}
        <rect x={r.x} y={r.y} width={r.width} height="1" />
    {/each}
</svg>
