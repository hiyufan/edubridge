/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        academic: {
          bg: '#F1F5F9',
          card: '#FFFFFF',
          sidebar: '#1E293B',
          'sidebar-hover': '#334155',
          primary: '#0F172A',
          accent: '#22C55E',
          'accent-hover': '#16A34A',
          amber: '#D97706',
          text: '#F8FAFC',
          'text-secondary': '#94A3B8',
          'text-dark': '#0F172A',
          'text-muted': '#64748B',
          border: '#334155',
          'border-light': '#E2E8F0',
          success: '#22C55E',
          warning: '#D97706',
          danger: '#EF4444',
          info: '#3B82F6',
        }
      },
      fontFamily: {
        heading: ['EB Garamond', 'Georgia', 'serif'],
        body: ['Crimson Text', 'Georgia', 'serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      spacing: {
        'sidebar': '240px',
        'sidebar-collapsed': '64px',
      },
      borderRadius: {
        'sm': '4px',
        'md': '8px',
        'lg': '12px',
        'xl': '16px',
      },
      transitionDuration: {
        '150': '150ms',
        '300': '300ms',
      }
    },
  },
  plugins: [],
}