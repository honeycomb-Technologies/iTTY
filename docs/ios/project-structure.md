# Project Structure

## Current `ios/` Layout

The repository now contains a Linux-generated scaffold that mirrors upstream Geistty into the intended iTTY layout:

```text
ios/
├── phase2-manifest.json
├── iTTY/
│   ├── Assets.xcassets/
│   ├── Resources/
│   └── Sources/
│       ├── App/
│       ├── Core/
│       ├── Models/
│       ├── Features/
│       └── Services/
├── iTTYTests/
│   └── UpstreamParity/
└── iTTYUITests/
    └── UpstreamParity/
```

## Notes

- `ios/iTTY/Sources/` is populated from the verified mapping in `ios/phase2-manifest.json`.
- iTTY-only product files now exist under `Models/`, `Services/Daemon/`, `Features/Machines/`, and `Features/SessionBrowser/`.
- `ios/iTTYTests/UpstreamParity/` and `ios/iTTYUITests/UpstreamParity/` are copied from upstream as parity references, not yet wired to a local Xcode target.
- `ios/iTTYTests/` also contains new iTTY-side tests for the daemon models/client, but they are not yet wired into a runnable Xcode test target.
- `_upstream/geistty/` remains the authoritative reference for the current Xcode project and any parity checks during migration.
