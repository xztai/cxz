import { defineConfig } from '@playwright/test';
import base from './playwright.config.js';

// A separate suite runs against its own fresh Compose stack, so browser tests
// exercise ordinary production rate limits without exhausting the auth suite.
export default defineConfig({ ...base, testDir: './tests/publications' });
