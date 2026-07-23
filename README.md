# React Native Fingerprint

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-react-native-fingerprint?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/releases)

Evaluates a cache-key template over your native inputs and exports BUNDLE_HASH_STRING, so subsequent cache steps can skip rebuilding native modules on JS-only changes.


<details>
<summary>Description</summary>

Evaluates a **cache-key template** — the same syntax the `restore-cache` / `save-cache` steps accept — and exports the result as **`BUNDLE_HASH_STRING`** so you can compute the key once and reuse it across restore, save, and build-gating.

The `key` input supports:
- `{{ checksum "package.json" "ios/Podfile.lock" "android/**/*.gradle" }}` — SHA-256 over one or more files; glob / `**` doublestar patterns are supported.
- `{{ getenv "SOME_ENV" }}` — interpolate an environment variable.
- `.OS`, `.Arch`, `.Branch`, `.CommitHash`, `.Workflow` — e.g. `{{ .OS }}-{{ .Arch }}-...`.

Use `BUNDLE_HASH_STRING` as the key for the subsequent `restore-cache` / `save-cache` steps, then gate the expensive native/bundle build on restore-cache's `BITRISE_CACHE_HIT` output so it only runs on a cache miss. On a hit, repack the new JS bundle into the cached app and re-sign.

**Capture every native input.** The fingerprint decides whether a previously compiled native app can be reused. If the key omits an input that affects the native build (a native lockfile, a Gradle file, a config plugin, a patch), a native-only change won't move the key — you'd restore and repack onto a **stale binary and ship a broken build**. The default below covers common React Native native inputs; extend it for your project (config plugins, `patches/`, extra native lockfiles).

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
| `key` | A cache-key template evaluated by the same engine as the `restore-cache` / `save-cache` steps.  Supported functions and variables: - `{{ checksum "<path>" ... }}` — SHA-256 of the listed files; supports glob and `**` doublestar patterns (e.g. `android/**/*.gradle`). Non-matching literal paths are skipped with a warning. - `{{ getenv "<ENV>" }}` — value of an environment variable. - `.OS`, `.Arch`, `.Branch`, `.CommitHash`, `.Workflow` — build metadata, e.g. `{{ .OS }}-{{ .Arch }}-{{ checksum "package.json" }}`.  The default fingerprints common React Native native inputs (JS lockfiles, `app.json` / `app.config.js`, `ios/Podfile.lock`, Android Gradle files). Deliberately excludes your JavaScript source, so a JS-only change reuses the cached native build. Extend it to cover anything else that affects your native build (config plugins, `patches/`, additional native lockfiles). The step fails if the template evaluates to an empty string.  | required | `{{ checksum "package.json" "package-lock.json" "yarn.lock" "pnpm-lock.yaml" "app.json" "app.config.js" "ios/Podfile.lock" "android/**/*.gradle" "android/gradle.properties" }}` |
| `verbose` | Enable to print (at debug level) the files matched by each `checksum` in the key. Useful for debugging cache-key changes and confirming your globs match what you expect.  | required | `false` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BUNDLE_HASH_STRING` | The evaluated `key` template. Always set on success (the step fails rather than exporting an empty key). Use it as the key for `restore-cache` / `save-cache` steps; gate the build on restore-cache's `BITRISE_CACHE_HIT` output. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
