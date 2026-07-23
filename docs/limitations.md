# Limitations & known gaps

This step fingerprints a project by taking **content checksums** of the files
you list in the `key` template. That is fast and dependency-light, but it is
*syntactic*: it reacts to file bytes, not to the semantic question "does this
change actually affect the compiled native app?". That yields two classes of
imperfection.

## Over-invalidation (safe, but wastes cache hits)

The key changes even though the native build would not have. You rebuild
natively when you did not strictly need to — never a wrong build, just a slower
one.

- **JS-only dependency changes.** Adding/removing/bumping a *pure-JavaScript*
  dependency changes `package.json` / the lockfile, so the key changes — even
  though no native module was affected.
- **Formatting / churn.** Key reordering, whitespace, or comments in a hashed
  file, and lockfile churn (integrity hashes, resolved URLs), all move the key
  even when the effective dependency set is unchanged.

## Under-capture (dangerous — must be configured correctly)

The key does **not** change even though the native build did. A cache hit then
reuses/repacks onto a stale native binary, which can ship a broken build. This
only happens with an incomplete or incorrect `key`; the default aims to avoid
it for common layouts, but you own the key.

- **Missing native inputs.** Anything that affects the native build but is not
  in `key` is invisible: non-standard native module directories, monorepo /
  workspace packages whose native code is not reflected in the root lockfile,
  `react-native.config.js`, iOS `Package.resolved` (SPM), `Gemfile.lock`,
  `.xcode.env`, custom config plugins, etc.
- **Semantic changes without a file change.** A native module pulled via a
  `file:`/git dependency or `npm link`, or a config plugin whose behavior
  changes transitively, may not move any hashed file.
- **Zero-match checksum with a literal prefix (subtle).** `checksum` returns an
  empty string (a warning, not an error) when its paths/globs match no files.
  The step fails on a *fully* empty key — but a key like
  `bundle-{{ checksum "wrong/path" }}` evaluates to the constant `bundle-`,
  which is non-empty and passes the guard while fingerprinting nothing. A typo
  or a layout the globs do not match can therefore produce a **constant key**
  that always hits the cache.

## Guidance

- Extend `key` to cover every input that affects *your* native build; do not
  assume the default is complete for your project.
- Enable `verbose` and confirm the logged file list matches what you expect —
  especially that glob patterns actually match files.
- Sanity-check that `BUNDLE_HASH_STRING` **changes** when you make a known
  native change (e.g. bump a Pod), and is **stable** across a JS-only change.
- Prefer keys without a misleading literal prefix on top of a single fragile
  glob, so a zero-match is more likely to surface as a failure.

## Future direction: semantic mode

The robust fix for the under-capture and over-invalidation classes is a
*semantic*, autolinking-aware fingerprint that hashes the resolved native
inputs (native module graph, native project files, config) rather than a
user-maintained file list. That is tracked as a separate follow-up.
