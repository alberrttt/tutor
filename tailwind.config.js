/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      fontFamily: {
        'space-mono': ['Space Mono', 'monospace'], // Add your custom font family
        'nunito': ['Nunito', 'sans-serif']
      },
    },
  },
  plugins: []
};
