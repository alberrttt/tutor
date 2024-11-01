import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import generateFonts from './generate-fonts';

export default defineConfig({
	plugins: [sveltekit(), generateFonts()],
	server: {
		// Allow serving files from the specified directory
		fs: {
			allow: ['./static']
		}
	}
});
