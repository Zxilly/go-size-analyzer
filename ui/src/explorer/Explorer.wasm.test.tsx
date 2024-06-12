import {it, vi} from "vitest";
import {render, screen, waitFor} from "@testing-library/react";
import {Explorer} from "./Explorer.tsx";

vi.mock("../../gsa.wasm?init", () => {
    return {
        default: async () => {
            return Promise.resolve(undefined)
        }
    }
})

vi.stubGlobal("Go", class {
    importObject = {}
    run = vi.fn(() => Promise.resolve())
    constructor() {
    }
})

it('Explorer should display error when loading fails', async () => {
    render(<Explorer/>)
    await waitFor(() => screen.getByText('Failed to load WebAssembly module'))
})