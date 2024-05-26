# UI for golang-size-analyzer

Uses code from [rollup-plugin-visualizer](https://github.com/btd/rollup-plugin-visualizer)

## Build

You should create a `data.json` with `gsa -f json --compact -o data.json <file>` and place it in the root of the project.
This file will be used as a data source when `pnpm dev` run.

```bash
pnpm install
pnpm run build:ui
```
