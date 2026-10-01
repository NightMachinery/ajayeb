// ESLint flat config. Not type-aware on purpose: a typed lint builds a full
// TypeScript program and costs too much memory on the laptop. svelte-check is
// the type gate.
import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import ts from 'typescript-eslint';
import svelteConfig from './svelte.config.js';

export default defineConfig(
  globalIgnores(['build/', '.svelte-kit/', 'node_modules/', 'src/lib/wire/*.gen.ts']),
  js.configs.recommended,
  ts.configs.recommended,
  svelte.configs.recommended,
  svelte.configs.prettier,
  {
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
    // TypeScript already reports undefined names, and knows about types.
    rules: { 'no-undef': 'off' }
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
    languageOptions: {
      parserOptions: {
        extraFileExtensions: ['.svelte'],
        parser: ts.parser,
        svelteConfig
      }
    }
  }
);
