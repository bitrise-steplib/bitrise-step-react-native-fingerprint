# React Native Build Fingerprint

A [Bitrise Step](https://devcenter.bitrise.io/) that computes a deterministic SHA-256 fingerprint of your dependency files and exports it so consequent `restore-cache` / `save-cache` steps can skip the native/bundle build when only JavaScript changed.

This is a lightweight, do-it-yourself pattern. For a fully managed, compilation-level remote cache across Gradle, Xcode (LLVM CAS) and C++, see [Bitrise Build Cache for React Native](https://bitrise.io/platform/build-cache/react-native).

## Inputs

| Key | Default | Description |
| --- | --- | --- |
| `file_paths` | `package.json`<br>`package-lock.json` | Newline-separated files whose contents determine the fingerprint. Blank lines and `#` comments are ignored. |
| `key_prefix` | _(empty)_ | Namespace prepended to the fingerprint (joined with `-`) to form `BUNDLE_HASH_STRING`. |
| `verbose` | `false` | Log the list of files being fingerprinted. |

## Outputs

| Env var | Description |
| --- | --- |
| `BUNDLE_HASH_STRING` | The dependency fingerprint, prefixed with `key_prefix` if set. Always set. Use as the key for `restore-cache` / `save-cache`; gate the build on restore-cache's `BITRISE_CACHE_HIT`. |


Runnable end-to-end example:

- Expo — <https://github.com/bitrise-silver/expo-build-fingerprint-demo>

## Development

```bash
go vet ./...
go build ./...
go test ./... -v
```

The hashing logic is pure functions with no Bitrise dependencies, so the unit tests run anywhere. Output export uses `envman`, which is present on Bitrise stacks. Cache hit/miss is determined by the following `restore-cache` step via its `BITRISE_CACHE_HIT` output — the fingerprint step itself makes no network calls.

## License

[MIT](LICENSE)
