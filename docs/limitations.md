# Limitations & known gaps

This step fingerprints a project by taking **content checksums** of the files
and folders in `paths` (folders hashed recursively), minus anything matched by
`ignore_paths`. That is fast and dependency-light, but it is *syntactic*: it
reacts to file bytes, not to the semantic question "does this change actually
affect the compiled native app?". That yields two classes of imperfection.

## Over-invalidation (safe, but wastes cache hits)

The key changes even though the native build would not have — you rebuild when
you did not strictly need to. Never a wrong build, just a slower one.

- **JS-only dependency changes.** Adding/removing/bumping a *pure-JavaScript*
  dependency changes `package.json` / the lockfile, so the key changes — even
  though no native module was affected.
- **Formatting / churn.** Key reordering, whitespace, or comments in a hashed
  file, and lockfile churn (integrity hashes, resolved URLs), move the key even
  when the effective dependency set is unchanged.

## Under-capture (dangerous — now much narrower)

The key does **not** change even though the native build did → a cache hit
reuses/repacks a stale native binary, which can ship a broken build. Recursive
folder hashing of `ios/` and `android/` closes most of this (any committed
native file change is caught), but residual cases remain:

- **Native changes that live only in `node_modules`.** A native module pulled
  via a `file:`/git dependency or `npm link`, or native code that changes
  without a lockfile change, is not hashed unless you add it to `paths`.
- **Inputs outside `project_dir`.** Monorepo/workspace packages or shared native
  code outside the project root are invisible unless explicitly listed.
- **Misconfiguration.** Removing `ios`/`android` from `paths`, a wrong
  `project_dir`, or an `ignore_paths` entry that is too broad can silently drop
  a real input.

## Why `ignore_paths` matters (correctness, not just speed)

Folder hashing REQUIRES ignores for two reasons:

1. **Cross-machine determinism.** Machine-specific files inside `ios/`/`android/`
   would make the hash differ between CI and local (cache never hits): e.g.
   `android/local.properties`, `ios/.xcode.env.local`, `**/xcuserdata`,
   `**/*.xcuserstate`.
2. **Build-output churn.** Regenerated each build: `ios/Pods`, `ios/build`,
   `android/build`, `android/app/build`, `android/.gradle`, `android/app/.cxx`.

The defaults cover these; extend them if your project has other volatile or
machine-specific paths under the hashed folders.

## Guidance

- Enable `verbose` and confirm the logged file list is what you expect.
- Sanity-check that `BUNDLE_HASH_STRING` **changes** on a known native change
  (e.g. bump a Pod) and is **stable** across a JS-only change.
- Extend `paths` to cover native inputs outside the standard `ios`/`android`
  layout; tighten `ignore_paths` if a volatile file is sneaking into the hash.

## Future direction: semantic mode

The robust fix for the over-invalidation class and the `node_modules`-only
native changes is a *semantic*, autolinking-aware fingerprint that hashes the
resolved native module graph rather than committed files. Tracked as a separate
follow-up.
