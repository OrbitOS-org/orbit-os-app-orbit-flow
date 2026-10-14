/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        // Theme colours come from the variables in src/assets/main.css.
        accent: {
          DEFAULT: 'rgb(var(--c-accent) / <alpha-value>)',
          dim: 'rgb(var(--c-accent-dim) / <alpha-value>)',
          muted: 'rgb(var(--c-accent) / 0.2)',
          subtle: 'rgb(var(--c-accent) / 0.05)',
        },
        terminal: {
          bg:           'rgb(var(--c-bg) / <alpha-value>)',
          surface:      'rgb(var(--c-surface) / <alpha-value>)',
          'surface-alt':'rgb(var(--c-surface-alt) / <alpha-value>)',
          border:       'rgb(var(--c-border) / <alpha-value>)',
          'border-light':'rgb(var(--c-border-light) / <alpha-value>)',
          text:         'rgb(var(--c-text) / <alpha-value>)',
          'text-dim':   'rgb(var(--c-text-dim) / <alpha-value>)',
          'text-bright':'rgb(var(--c-text-bright) / <alpha-value>)',
        },
        status: {
          success: 'rgb(var(--c-success) / <alpha-value>)',
          warning: 'rgb(var(--c-warning) / <alpha-value>)',
          error:   'rgb(var(--c-error) / <alpha-value>)',
        },
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', '"SF Pro Text"', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['"SF Mono"', 'ui-monospace', 'IBM Plex Mono', 'Menlo', 'Consolas', 'monospace'],
      },
      borderRadius: {
        // iOS-like corners.
        DEFAULT: '8px', none: '0px', sm: '6px', md: '8px',
        lg: '10px', xl: '12px', '2xl': '14px', '3xl': '18px',
        full: '9999px',
      },
    },
  },
  plugins: [],
};
