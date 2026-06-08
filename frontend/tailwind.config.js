/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: [
          'Inter',
          '-apple-system',
          'system-ui',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'sans-serif',
        ],
      },
      colors: {
        woot: {
          50: '#e5f0ff',
          100: '#cce0ff',
          200: '#99c2ff',
          300: '#66a3ff',
          400: '#3385ff',
          500: '#1f93ff',
          600: '#0077eb',
          700: '#005bb8',
          800: '#003f85',
          900: '#002352',
        },
        n: {
          'solid-1': '#ffffff',
          'solid-2': '#f8fafc',
          'solid-3': '#f1f5f9',
          weak: '#e2e8f0',
          strong: '#cbd5e1',
          'slate-11': '#64748b',
          'slate-12': '#0f172a',
          brand: '#1f93ff',
        },
      },
      boxShadow: {
        card: '0 1px 2px rgba(15, 23, 42, 0.06)',
      },
    },
  },
  plugins: [],
};
