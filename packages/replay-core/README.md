# @pinconsole/replay-core

Fork of [rrweb](https://github.com/rrweb-io/rrweb) v2.0.0-alpha.20 snapshot/record/replay modules, pruned and enhanced for pinconsole.

## Why fork

- Eliminate Svelte-based rrweb-player dependency
- Enable nodeID-based cross-end element addressing
- Remove unused features (canvas recording, console plugin, packer)
- Simplify to single-package structure for easier maintenance

## Source

Forked from rrweb-io/rrweb@`77e20807` (v2.0.0-alpha.20).
See [NOTICE](./NOTICE) for license and attribution.

## Structure

```
src/
├── index.ts        # Main entry (re-exports record + Replayer + types)
├── rrweb-types.ts  # Package-level types (recordOptions, ReplayPlugin, etc.)
├── types/          # Shared types (forked from @rrweb/types)
├── dom-utils/      # DOM utilities (forked from @rrweb/utils)
├── snapshot/       # DOM snapshot/rebuild (forked from rrweb-snapshot)
├── record/         # Recording (forked from rrweb/src/record)
├── replay/         # Replay/player (forked from rrweb/src/replay)
└── rrdom/          # Virtual DOM (forked from rrdom)
```
