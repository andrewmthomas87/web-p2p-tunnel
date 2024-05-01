import html, { makeHtmlAttributes } from '@rollup/plugin-html';
import replace from '@rollup/plugin-replace';
import typescript from '@rollup/plugin-typescript';
import 'dotenv/config';
import { readFileSync } from 'fs';
import { resolve } from 'path';
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
        template: fileTemplate('src/tunnel.html'),
        meta: [
          { charset: 'utf-8' },
          { name: 'viewport', content: 'width=device-width,initial-scale=1,maximum-scale=1' },
        ],
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

function fileTemplate(relativePath) {
  const path = resolve(process.cwd(), relativePath);
  const data = readFileSync(path, 'utf8');

  // Adapted from https://github.com/rollup/plugins/blob/9e7f576f33e26d65e9f2221d248a2e000923e03f/packages/html/src/index.ts#L29
  return ({ attributes, files, meta, publicPath, title }) => {
    const scripts = (files.js || [])
      .map(({ fileName }) => {
        const attrs = makeHtmlAttributes(attributes.script);
        return `<script src="${publicPath}${fileName}"${attrs}></script>`;
      })
      .join('\n');

    const links = (files.css || [])
      .map(({ fileName }) => {
        const attrs = makeHtmlAttributes(attributes.link);
        return `<link href="${publicPath}${fileName}" rel="stylesheet"${attrs}>`;
      })
      .join('\n');

    const metas = meta
      .map((input) => {
        const attrs = makeHtmlAttributes(input);
        return `<meta${attrs}>`;
      })
      .join('\n');

    return `
<!doctype html>
<html${makeHtmlAttributes(attributes.html)}>
  <head>
    ${metas}
    <title>${title}</title>
    ${links}
  </head>
  <body>
    ${data}
    ${scripts}
  </body>
</html>`;
  };
}
