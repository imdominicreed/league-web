/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // League client colors
        'lol-dark': '#010a13',
        'lol-dark-blue': '#0a1428',
        'lol-gold': '#c8aa6e',
        'lol-gold-dark': '#785a28',
        'lol-gold-light': '#f0e6d2',
        'lol-blue-accent': '#0ac8b9',
        'lol-gray': '#1e2328',
        'lol-border': '#463714',
        // Team colors
        'blue-team': '#0088cc',
        'red-team': '#cc3333',
        // Legacy aliases
        'lol-blue': '#0ac8b9',
        'blue-side': '#0088cc',
        'red-side': '#cc3333',
      },
      fontFamily: {
        'beaufort': ['Cinzel', 'serif'],
      },
    },
  },
  plugins: [],
}
