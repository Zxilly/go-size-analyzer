import {fireEvent, render} from '@testing-library/react';
import {assert, expect, test} from 'vitest';
import {Tooltip} from './Tooltip.tsx';
import {Entry, EntryChildren, type EntryType} from "./tool/entry.ts";

function getTestNode(): Entry {
    return {
        getChildren(): EntryChildren[EntryType] {
            return [];
        },
        getID(): number {
            return 1;
        }, getName(): string {
            return "test";
        }, getSize(): number {
            return 12345;
        }, toString(): string {
            return "test content";
        }, getType(): EntryType {
            return "unknown"
        }, getURLSafeName(): string {
            return "test";
        }
    }
}

test('Tooltip should render correctly when visible', () => {
    const {getByText} = render(<Tooltip visible={true}
                                        node={getTestNode()}/>);
    expect(getByText('test')).toBeInTheDocument();
    expect(getByText('test content')).toBeInTheDocument();
});

test('Tooltip should not render when not visible', () => {
    const r = render(<Tooltip visible={false}
                              node={getTestNode()}/>);
    // should have tooltip-hidden class
    expect(r.container.querySelector('.tooltip-hidden')).not.toBeNull();
});

test('Tooltip should update position on mouse move', () => {
    const {getByText} = render(<Tooltip visible={true} node={getTestNode()}/>);
    fireEvent.mouseMove(document, {clientX: 100, clientY: 100});
    const tooltip = getByText('test').parentElement;
    if (!tooltip) {
        assert.isNotNull(tooltip);
        return;
    }

    expect(tooltip.style.left).toBe('110px');
    expect(tooltip.style.top).toBe('130px');
});