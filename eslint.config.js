import { defineConfig } from 'eslint/config'
import js from '@eslint/js'
import ts from 'typescript-eslint'
import svelte from 'eslint-plugin-svelte'
import prettier from 'eslint-config-prettier'
import globals from 'globals'
import svelteConfig from './svelte.config.js'

export default defineConfig(
  js.configs.recommended,
  ...ts.configs.recommended,
  svelte.configs.recommended,
  prettier,
  svelte.configs.prettier,
  {
    languageOptions: {
      globals: { ...globals.browser, ...globals.node }
    }
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
        extraFileExtensions: ['.svelte'],
        svelteConfig
      }
    }
  },
  {
    rules: {
      // Locale prefixes are applied by `reroute` + `localizedPath`, not `$app/paths`.
      'svelte/no-navigation-without-resolve': 'off'
    }
  },
  {
    ignores: [
      'build/',
      '.svelte-kit/',
      'dist/',
      'kun-love-ren/',
      'node_modules/',
      'prisma/generated/',
      '**/*.cjs'
    ]
  }
)
