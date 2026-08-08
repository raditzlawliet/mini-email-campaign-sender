import { mount } from "svelte";
import App from "./App.svelte";
import "./app.css";

// Wails' built-in edge resize breaks when a DOM scrollbar occupies the window
// edge (the 6px resize zone is swallowed by the scrollbar strip). Window
// resizing is handled by WindowResizeHandles instead, which works on all
// platforms. See wailsapp/wails#4680.
if (window.wails?.flags) {
    window.wails.flags.enableResize = false;
}

const app = mount(App, {
  target: document.getElementById("app"),
});

export default app;
