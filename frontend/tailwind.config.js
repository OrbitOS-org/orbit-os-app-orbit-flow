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
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['IBM Plex Mono', 'Consolas', 'Monaco', 'Courier New', 'monospace'],
      },
      borderRadius: {
        DEFAULT: '3px', none: '0px', sm: '2px', md: '4px',
        lg: '6px', xl: '8px', '2xl': '12px', '3xl': '16px',
        full: '9999px',
      },
    },
  },
  plugins: [],
};
