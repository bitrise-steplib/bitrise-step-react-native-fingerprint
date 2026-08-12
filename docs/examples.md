### Examples

Compute the fingerprint with the default React Native inputs and use it as a cache key:

```yaml
- react-native-fingerprint:
    inputs:
    - key_prefix: android
- restore-cache:
    inputs:
    - key: $BUNDLE_HASH_STRING
```

Skip the native rebuild when only JavaScript changed — fingerprint, restore, gate the build on the cache hit, save on a miss:

```yaml
- react-native-fingerprint:
    inputs:
    - key_prefix: android
- restore-cache:
    inputs:
    - key: $BUNDLE_HASH_STRING
- android-build:
    run_if: '{{not (enveq "BITRISE_CACHE_HIT" "exact")}}'
    inputs:
    - project_location: ./android
    - variant: release
- save-cache:
    run_if: '{{not (enveq "BITRISE_CACHE_HIT" "exact")}}'
    inputs:
    - key: $BUNDLE_HASH_STRING
    - paths: android/app/build/outputs/apk/release/app-release.apk
- script:
    title: Repack the new JS bundle into the cached app
    run_if: '{{enveq "BITRISE_CACHE_HIT" "exact"}}'
    inputs:
    - content: |-
        #!/usr/bin/env bash
        set -euo pipefail
        # The cached APK was restored — only the JS changed, so bundle it and repack:
        npx react-native bundle --platform android --dev false \
          --entry-file index.js --bundle-output index.android.bundle
        # Replace assets/index.android.bundle inside the restored APK,
        # then zipalign and re-sign (apksigner) before deploying.
```

Fingerprint a project in a subdirectory, with native inputs beyond the defaults:

```yaml
- react-native-fingerprint:
    inputs:
    - project_dir: ./mobile-app
    - key_prefix: ios
    - path_list: |-
        package.json
        yarn.lock
        app.json
        patches/
        ios/
        android/
        Gemfile.lock
        modules/my-native-module/
```

Log every file that contributes to the fingerprint (for verifying `path_list` / `ignore_path_list`):

```yaml
- react-native-fingerprint:
    inputs:
    - verbose: "true"
```
