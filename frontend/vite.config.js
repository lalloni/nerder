import {defineConfig} from 'vite';

// 'mpa' makes Vite answer unknown paths (such as /fonts/...) with 404 instead
// of index.html, so Wails can hand them to the Go font handler in dev mode.
export default defineConfig({appType: 'mpa'});
