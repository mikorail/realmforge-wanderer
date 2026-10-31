import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
	server: { port: 5195, strictPort: true },
	plugins: [svelte({ compilerOptions: { runes: true } })]
});
