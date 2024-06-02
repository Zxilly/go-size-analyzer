// @vitest-environment jsdom

import {fireEvent, render} from '@testing-library/react';
import {FileSelector} from './file_selector.tsx';
import {assert, expect, test, vi} from 'vitest';

test('FileSelector should render correctly', () => {
    const mockHandler = vi.fn();
    const {getByText} = render(<FileSelector handler={mockHandler}/>);
    expect(getByText('Select file')).toBeInTheDocument();
});

test('FileSelector should call handler when file size is within limit', () => {
    const mockHandler = vi.fn();
    const {getByLabelText} = render(<FileSelector handler={mockHandler}/>);
    const file = new File(['dummy content'], 'dummy.txt', {type: 'text/plain'});
    fireEvent.change(getByLabelText('Select file'), {target: {files: [file]}});
    expect(mockHandler).toHaveBeenCalledWith(file);
});

test('FileSelector should not call handler when file size exceeds limit', () => {
    const mockHandler = vi.fn();
    const {getByLabelText} = render(<FileSelector handler={mockHandler}/>);
    const file = new File(["0".repeat(1024 * 1024 * 31)], 'dummy.txt', {type: 'text/plain'});
    assert(file.size > 1024 * 1024 * 30);

    fireEvent.change(getByLabelText('Select file'), {target: {files: [file]}});
    expect(mockHandler).not.toHaveBeenCalled();
});

test('FileSelector should call handler when file size exceeds limit and user chooses to continue', () => {
    const mockHandler = vi.fn();
    const {getByLabelText, getByText} = render(<FileSelector handler={mockHandler}/>);
    const file = new File(["0".repeat(1024 * 1024 * 31)], 'dummy.txt', {type: 'text/plain'});
    assert(file.size > 1024 * 1024 * 30);

    fireEvent.change(getByLabelText('Select file'), {target: {files: [file]}});
    fireEvent.click(getByText('Continue'));
    expect(mockHandler).toHaveBeenCalledWith(file);
});