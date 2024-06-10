import { expect, test } from 'vitest';
import { aligner } from './aligner.ts';

test('aligner should correctly add and align strings', () => {
    const al = new aligner();
    al.add('short', 'post1');
    al.add('a bit longer', 'post2');
    expect(al.toString()).toBe('short        post1\n' +
        'a bit longer post2');
});

test('aligner should handle empty pre string', () => {
    const al = new aligner();
    al.add('', 'post1');
    al.add('a bit longer', 'post2');
    expect(al.toString()).toBe('             post1\n' +
        'a bit longer post2');
});

test('aligner should handle empty post string', () => {
    const al = new aligner();
    al.add('short', '');
    al.add('a bit longer', 'post2');
    expect(al.toString()).toBe('short        \na bit longer post2');
});

test('aligner should handle empty pre and post strings', () => {
    const al = new aligner();
    al.add('', '');
    al.add('a bit longer', 'post2');
    expect(al.toString()).toBe('             \n' +
        'a bit longer post2');
});

test('aligner should handle no added strings', () => {
    const al = new aligner();
    expect(al.toString()).toBe('');
});