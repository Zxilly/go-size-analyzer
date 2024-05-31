// @vitest-environment jsdom

import {formatBytes, loadDataFromEmbed, title, trimPrefix} from './utils.ts';
import {expect, test} from 'vitest';


test('loadDataFromEmbed should throw error when data is not parsable', () => {
    document.body.innerHTML = '<div id="data">unparsable data</div>';
    expect(() => loadDataFromEmbed()).toThrow();
});

test('formatBytes should correctly format bytes into human readable format', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(1024)).toBe('1 KB');
    expect(formatBytes(1048576)).toBe('1 MB');
});

test('title should capitalize the first letter of the string', () => {
    expect(title('hello')).toBe('Hello');
    expect(title('world')).toBe('World');
});

test('trimPrefix should remove the prefix from the string', () => {
    expect(trimPrefix('HelloWorld', 'Hello')).toBe('World');
    expect(trimPrefix('HelloWorld', 'World')).toBe('HelloWorld');
});