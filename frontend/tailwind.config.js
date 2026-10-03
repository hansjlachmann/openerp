/** @type {import('tailwindcss').Config} */

// Business Central color palette, sampled from docs/BC-Screenshot-LightMode.png and
// docs/BC-Screenshot-DarkMode.png. Components use Tailwind's gray/blue/primary classes,
// so those scales are retuned here to BC values instead of editing every component.
//
// Neutrals: each step serves its light-mode role (text/borders on white) and its
// dark-mode role (surfaces/borders/text on #121212):
//   50  #f7f7f7  dark: primary text           100 #f2f2f3  light: hover background
//   200 #e5e7e9  light: neutral tile, borders 300 #d3d6da  light: divider
//   400 #a4b0c4  dark: secondary text         500 #737d8a  muted text
//   600 #505c6d  light: secondary text        700 #303032  dark: separators, inputs
//   800 #1e1e1e  dark: raised surface         900 #121212  dark: page background
const bcGray = {
	50: '#f7f7f7',
	100: '#f2f2f3',
	200: '#e5e7e9',
	300: '#d3d6da',
	400: '#a4b0c4',
	500: '#737d8a',
	600: '#505c6d',
	700: '#303032',
	800: '#1e1e1e',
	900: '#121212',
	950: '#0a0a0a'
};

// BC teal accent (BC has no blue accent): 400 = dark-mode links, 500 = tiles,
// 600 = light-mode links/actions, 900 = dark-mode notification bar
const bcTeal = {
	50: '#e6f3f4',
	100: '#cce8ea',
	200: '#99d1d4',
	300: '#66b9bf',
	400: '#37a1a5',
	500: '#00838f',
	600: '#008489',
	700: '#006e72',
	800: '#00585c',
	900: '#003a3e',
	950: '#002a2d'
};

export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	darkMode: 'class',
	theme: {
		extend: {
			colors: {
				gray: bcGray,
				blue: bcTeal,
				primary: bcTeal,
				nav: {
					blue: '#282828', // BC app bar (same in light and dark mode)
					lightblue: '#008489' // BC accent
				}
			},
			fontFamily: {
				sans: ['Segoe UI', 'system-ui', 'sans-serif']
			}
		}
	},
	plugins: []
};
