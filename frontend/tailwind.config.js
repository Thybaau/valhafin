/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: {
          primary: '#0a0a0a',
          secondary: '#1a1a1a',
          tertiary: '#2a2a2a',
        },
        text: {
          primary: '#ffffff',
          secondary: '#b0b0b0',
          muted: '#6b6b6b',
        },
        accent: {
          primary: '#3b82f6',
          hover: '#2563eb',
          light: '#60a5fa',
        },
        success: '#10b981',
        error: '#ef4444',
        warning: '#f59e0b',
      },
      borderRadius: {
        sm: '0.375rem',
        md: '0.5rem',
        lg: '0.75rem',
      },
      boxShadow: {
        sm: '0 1px 2px 0 rgba(0, 0, 0, 0.5)',
        md: '0 4px 6px -1px rgba(0, 0, 0, 0.5)',
        lg: '0 10px 15px -3px rgba(0, 0, 0, 0.5)',
      },
    },
  },
  plugins: [],
}
