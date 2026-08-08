<script>
    import { onMount } from "svelte";
    import {
        WindowGetSize,
        WindowGetPosition,
        WindowSetSize,
        WindowSetPosition,
        WindowIsMaximised,
        EventsOn,
        EventsOff,
    } from "../wailsjs/runtime/runtime";

    // Keep in sync with MinWidth/MinHeight in main.go
    const MIN_W = 800;
    const MIN_H = 600;

    let isMaximized = $state(false);

    // Active resize drag state
    let drag = null; // { edge, sx, sy, sw, sh, wx, wy }
    // Queued size/position change, applied once per animation frame
    let pending = null; // { x, y, w, h }
    let rafId = null;

    function applyPending() {
        rafId = null;
        if (!pending) return;
        const p = pending;
        pending = null;
        if (p.x !== undefined) WindowSetPosition(p.x, p.y);
        WindowSetSize(p.w, p.h);
    }

    function scheduleApply() {
        if (rafId) return;
        rafId = requestAnimationFrame(applyPending);
    }

    async function refreshMaximized() {
        try {
            isMaximized = await WindowIsMaximised();
        } catch {
            // runtime unavailable (plain browser dev) - keep current state
        }
    }

    function onPointerDown(edge, e) {
        if (isMaximized) return;
        e.preventDefault();
        e.currentTarget.setPointerCapture(e.pointerId);
        Promise.all([WindowGetSize(), WindowGetPosition(), WindowIsMaximised()])
            .then(([size, pos, maximised]) => {
                if (maximised) {
                    drag = null;
                    return;
                }
                drag = {
                    edge,
                    sx: e.screenX,
                    sy: e.screenY,
                    sw: size.w,
                    sh: size.h,
                    wx: pos.x,
                    wy: pos.y,
                };
            })
            .catch(() => {
                drag = null;
            });
    }

    function onPointerMove(e) {
        if (!drag) return;
        const d = drag;
        const dx = e.screenX - d.sx;
        const dy = e.screenY - d.sy;

        let w = d.sw;
        let h = d.sh;
        let x = d.wx;
        let y = d.wy;

        if (d.edge.includes("e")) w = d.sw + dx;
        if (d.edge.includes("s")) h = d.sh + dy;
        if (d.edge.includes("w")) w = d.sw - dx;
        if (d.edge.includes("n")) h = d.sh - dy;

        w = Math.max(w, MIN_W);
        h = Math.max(h, MIN_H);
        // Keep the opposite edge fixed when clamping from the left/top
        if (d.edge.includes("w")) x = d.wx + (d.sw - w);
        if (d.edge.includes("n")) y = d.wy + (d.sh - h);

        pending =
            d.edge.includes("w") || d.edge.includes("n")
                ? { x, y, w, h }
                : { w, h };
        scheduleApply();
    }

    function onPointerUp(e) {
        if (!drag) return;
        drag = null;
        if (rafId) {
            cancelAnimationFrame(rafId);
            rafId = null;
        }
        applyPending();
        try {
            e.currentTarget.releasePointerCapture(e.pointerId);
        } catch {
            // capture may already be released
        }
    }

    function handleProps(edge) {
        return {
            onpointerdown: (e) => onPointerDown(edge, e),
            onpointermove: onPointerMove,
            onpointerup: onPointerUp,
            onpointercancel: onPointerUp,
        };
    }

    onMount(() => {
        refreshMaximized();
        EventsOn("wails:maximised", () => {
            isMaximized = true;
        });
        EventsOn("wails:unmaximised", () => {
            isMaximized = false;
        });
        // Fallback for platforms without maximize events (e.g. Linux)
        let timer;
        const onWinResize = () => {
            clearTimeout(timer);
            timer = setTimeout(refreshMaximized, 150);
        };
        window.addEventListener("resize", onWinResize);
        return () => {
            EventsOff("wails:maximised");
            EventsOff("wails:unmaximised");
            window.removeEventListener("resize", onWinResize);
        };
    });
</script>

{#if !isMaximized}
    <div class="pointer-events-none fixed inset-0 z-60 select-none">
        <!-- edges -->
        <div
            {...handleProps("n")}
            class="pointer-events-auto absolute top-0 right-0 left-0 h-1.5 cursor-n-resize"
        ></div>
        <div
            {...handleProps("s")}
            class="pointer-events-auto absolute right-0 bottom-0 left-0 h-1.5 cursor-s-resize"
        ></div>
        <div
            {...handleProps("e")}
            class="pointer-events-auto absolute top-0 right-0 bottom-0 w-1.5 cursor-e-resize"
        ></div>
        <div
            {...handleProps("w")}
            class="pointer-events-auto absolute top-0 bottom-0 left-0 w-1.5 cursor-w-resize"
        ></div>
        <!-- corners -->
        <div
            {...handleProps("nw")}
            class="pointer-events-auto absolute top-0 left-0 size-2.5 cursor-nwse-resize"
        ></div>
        <div
            {...handleProps("ne")}
            class="pointer-events-auto absolute top-0 right-0 size-2.5 cursor-nesw-resize"
        ></div>
        <div
            {...handleProps("sw")}
            class="pointer-events-auto absolute bottom-0 left-0 size-2.5 cursor-nesw-resize"
        ></div>
        <div
            {...handleProps("se")}
            class="pointer-events-auto absolute right-0 bottom-0 size-2.5 cursor-nwse-resize"
        ></div>
    </div>
{/if}
