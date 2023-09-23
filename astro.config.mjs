import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'SCount',
			description: 'Design and documentation for SCount',
			social: {
				github: 'https://github.com/manojnakp/scount',
			},
			sidebar: [
			],
			lastUpdated: true,
			editLink: {
				baseUrl: 'https://github.com/manojnakp/scount/edit/main',
			},
		}),
	],
	site: 'https://manojnakp.github.io',
	base: '/scount',
});
