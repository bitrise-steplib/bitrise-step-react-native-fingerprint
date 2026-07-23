# React Native Fingerprint

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-react-native-fingerprint?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/releases)

Fingerprints your React Native native inputs (native project folders, lockfiles, config, patches) and exports BUNDLE_HASH_STRING, so subsequent cache steps can skip rebuilding native modules on JS-only changes.


<details>
<summary>Description</summary>

Computes a deterministic SHA-256 **fingerprint** of the inputs that determine your native build and exports it as **`BUNDLE_HASH_STRING`**, so you can key `restore-cache` / `save-cache` off it and skip the expensive native build when only JavaScript changed.

It hashes the files and folders in `paths` (folders are hashed recursively), while excluding anything matched by `ignore_paths`. The default `paths` cover the common React Native native inputs — JS lockfiles, `app.json` / `app.config.js`, `patch-package` patches, and the `ios/` and `android/` native projects. It deliberately does not hash your JavaScript source, so a JS-only change reuses the cached native build.

`ignore_paths` is important for **correctness**, not just speed: it excludes build outputs that change every run (`ios/Pods`, `android/build`, …) and machine-specific files that would otherwise make the fingerprint differ between CI and local (`android/local.properties`, Xcode user state, …).

Use `BUNDLE_HASH_STRING` as the key for `restore-cache` / `save-cache`, then gate the native/bundle build on restore-cache's `BITRISE_CACHE_HIT` output. On a hit, repack the new JS bundle into the cached app and re-sign.

The step is skippable: if fingerprinting can't produce a key (or fails), it exports an **empty** `BUNDLE_HASH_STRING` — a cache miss that safely forces a rebuild — rather than blocking your pipeline.

**Capture every native input.** The fingerprint decides whether a previously compiled native app can be reused. If `paths`/`ignore_paths` omit an input that affects the native build (a native module dir, a config plugin, an extra lockfile), a native change may not move the key and you could repack onto a **stale binary**. Extend `paths` for your project when needed.

This is a lightweight, do-it-yourself pattern built on free Bitrise cache steps. For a fully managed, compilation-level remote cache across Gradle, Xcode (LLVM CAS) and C++, see Bitrise Build Cache for React Native: https://bitrise.io/platform/build-cache/react-native

</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `project_dir` | The React Native project root. `paths` and `ignore_paths` are interpreted relative to this directory, and the fingerprint uses each file's path relative to it (so the result is independent of where the project is checked out).  | required | `.` |
| `paths` | One entry per line, relative to `project_dir`: - a **file** — e.g. `package.json` - a **folder** — write it with a trailing slash, e.g. `ios/` (hashed recursively) - a **glob** — a `*` / `**` doublestar pattern, e.g. `android/**/*.gradle`  The trailing slash is a readability convention — the step detects each entry's type automatically, so `ios` and `ios/` behave the same. Missing entries are skipped with a warning. Blank lines and lines starting with `#` are ignored.  The default covers common React Native native inputs. Extend it with anything else that affects your native build (custom native module directories, additional lockfiles, `Gemfile.lock`, etc.).  | required | `package.json package-lock.json yarn.lock pnpm-lock.yaml app.json app.config.js patches/ ios/ android/` |
| `ignore_paths` | Entries (relative to `project_dir`) excluded from hashing, matched as directory prefixes or `**` doublestar globs. This list matters for correctness: it excludes per-build outputs (`ios/Pods`, `android/build`, …) and machine-specific files (`android/local.properties`, Xcode user state) that would otherwise change the fingerprint between machines and prevent cache hits. Blank lines and `#` comments are ignored.  |  | `ios/Pods ios/build ios/DerivedData ios/.xcode.env.local android/build android/app/build android/.gradle android/app/.cxx android/local.properties **/xcuserdata **/*.xcuserstate **/*.log **/.DS_Store node_modules .git` |
| `key_prefix` | When set, it is prepended to the fingerprint to form `BUNDLE_HASH_STRING` (e.g. `ios-<fingerprint>`). Useful for namespacing the cache key per platform or workflow so different builds don't collide.  |  |  |
| `verbose` | Enable to print (at debug level) each file included in the fingerprint. Useful for confirming your `paths` and `ignore_paths` match what you expect.  | required | `false` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BUNDLE_HASH_STRING` | The computed native-input fingerprint, prefixed with `key_prefix` if set. **Empty** when no inputs matched — treat an empty value as a cache miss and rebuild the app. Use it as the key for `restore-cache` / `save-cache` steps; gate the build on restore-cache's `BITRISE_CACHE_HIT` output. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
