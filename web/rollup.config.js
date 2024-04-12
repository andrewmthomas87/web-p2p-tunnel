import html from '@rollup/plugin-html';
import replace from '@rollup/plugin-replace';
import typescript from '@rollup/plugin-typescript';
import 'dotenv/config';
import { defineConfig } from 'rollup';

const isProduction = process.env.NODE_ENV === 'production';

export default defineConfig([
  {
    input: 'src/main.ts',
    output: {
      dir: 'dist',
      entryFileNames: '[name]-[hash].js',
    },
    plugins: [
      typescript({ tsconfig: 'src/tsconfig.json' }),
      html({ title: 'web-p2p-tunnel' }),
      isProduction && (await import('@rollup/plugin-terser')).default(),
    ],
  },
  {
    input: 'src/tunnel.ts',
    output: {
      dir: 'dist',
      entryFileNames: '[name]-[hash].js',
    },
    plugins: [
      replace({
        'import.meta.env.PUBLIC_SIGNALING_SERVER_URL': JSON.stringify(
          process.env.PUBLIC_SIGNALING_SERVER_URL,
        ),
      }),
      typescript({ tsconfig: 'src/tsconfig.json' }),
      html({
        fileName: 'tunnel.html',
        title: 'web-p2p-tunnel',
      }),
      isProduction && (await import('@rollup/plugin-terser')).default(),
    ],
  },
  {
    input: 'sw/sw.ts',
    output: { dir: 'dist' },
    plugins: [
      typescript({ tsconfig: 'sw/tsconfig.json' }),
      isProduction && (await import('@rollup/plugin-terser')).default(),
    ],
  },
]);
