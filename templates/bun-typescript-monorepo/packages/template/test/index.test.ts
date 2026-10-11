import { expect, test } from 'bun:test';

import { createGreeting } from '../src/index';

test('creates a default greeting', () => {
  expect(createGreeting()).toBe('Hello, World!');
});

test('creates a greeting for a provided name', () => {
  expect(createGreeting('Project')).toBe('Hello, Project!');
});
