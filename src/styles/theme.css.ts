import { createGlobalTheme } from '@vanilla-extract/css';

export const vars = createGlobalTheme(':root', {
  colors: {
    primary: '#3b82f6',
    background: '#0f172a',
    surface: '#1e293b',
    text: '#f8fafc',
    textMuted: '#94a3b8',
    danger: '#ef4444',
    warning: '#f59e0b',
    success: '#10b981'
  },
  spacing: {
    small: '8px',
    medium: '16px',
    large: '24px',
    xlarge: '32px'
  },
  typography: {
    fontFamily: '"Inter", sans-serif',
    fontSize: {
      small: '12px',
      base: '16px',
      large: '20px',
      xlarge: '24px'
    }
  }
});
