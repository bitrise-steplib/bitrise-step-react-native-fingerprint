# React Native Fingerprint

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-react-native-fingerprint?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/releases)

Fingerprints your dependency files and exports BUNDLE_HASH_STRING so subsequent cache steps can skip rebuilding native modules on JS-only changes.


<details>
<summary>Description</summary>

Computes a deterministic SHA-256 **fingerprint** of the files you list (e.g. `package.json`, `package-lock.json`, `Podfile.lock`), optionally namespaced with `key_prefix`, and exports it as **`BUNDLE_HASH_STRING`**.

Use `BUNDLE_HASH_STRING` as the key for the subsequent `restore-cache` / `save-cache` steps, then gate the expensive native/bundle build on restore-cache's `BITRISE_CACHE_HIT` output so it only runs on a cache miss.

This is a lightweight, do-it-yourself pattern built entirely on free Bitrise cache steps. For a fully managed, compilation-level remote cache across Gradle, Xcode (LLVM CAS) and C++, see Bitrise Build Cache for React Native: https://bitrise.io/platform/build-cache/react-native

</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `file_paths` | One path per line. The step reads the contents of each file and derives a single, order-independent SHA-256 fingerprint. Typical inputs are `package.json`, `package-lock.json` / `yarn.lock` / `pnpm-lock.yaml`, and native lockfiles such as `ios/Podfile.lock`. Blank lines and lines starting with `#` are ignored.  | required | `package.json package-lock.json` |
| `key_prefix` | Optional. When set, it's prepended to the fingerprint (joined with a `-`) to form `BUNDLE_HASH_STRING`. Useful for namespacing the cache key per platform/workflow so different builds don't collide on the same key.  |  |  |
| `verbose` | Enable to print each file that contributes to the fingerprint. Useful for debugging cache-key changes.  | required | `false` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BUNDLE_HASH_STRING` | The dependency fingerprint, prefixed with `key_prefix` if set. Always set on success. Use it as the key for `restore-cache` / `save-cache` steps; gate the build on restore-cache's `BITRISE_CACHE_HIT` output. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
