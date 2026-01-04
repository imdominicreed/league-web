/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'lol-blue': '#0AC8B9',
        'lol-gold': '#C89B3C',
        'lol-dark': '#010A13',
        'lol-darker': '#000000',
        'blue-side': '#0066CC',
        'red-side': '#CC0033',
      },
    },
  },
  plugins: [],
}
