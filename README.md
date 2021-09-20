# Install missing Android SDK components

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-install-missing-android-tools?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-install-missing-android-tools/releases)

Install Android SDK components that are required for the app.

<details>
<summary>Description</summary>


This Step makes sure that required Android SDK components (platforms and build-tools) are installed. To do so, the Step runs the `gradlew dependencies` command. 

If the Android Plugin for Gradle version is 2.2.0 or higher, the plugin will download and install the missing components during the Gradle command.  
Otherwise the command fails and the Step parses the command's output to determine which SDK components are missing and installs them.

### Configuring the Step

1. Set the path of the `gradlew` file.

   The default value is that of the $PROJECT_LOCATION Environment Variable.

1. If you use an Android NDK in your app, set its revision in the **NDK Revision** input.

### Troubleshooting

If the Step fails, check that your repo actually contains a `gradlew` file. Without the Gradle wrapper, this Step won't work.

### Useful links

[Installing an additional Android SDK package](https://devcenter.bitrise.io/tips-and-tricks/android-tips-and-tricks/#how-to-install-an-additional-android-sdk-package)

### Related Steps

* [Android SDK Update](https://www.bitrise.io/integrations/steps/android-sdk-update)
* [Install React Native](https://www.bitrise.io/integrations/steps/install-react-native)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `gradlew_path` | Using a Gradle Wrapper (gradlew) is required, as the wrapper is what makes sure that the right Gradle version is installed and used for the build. __You can find more information about the Gradle Wrapper (gradlew), and about how you can generate one (if you would not have one already)__ in the official guide at: [https://docs.gradle.org/current/userguide/gradle_wrapper.html](https://docs.gradle.org/current/userguide/gradle_wrapper.html).  **The path should be relative** to the repository root, for example: `./gradlew`, or if it's in a sub directory: `./sub/dir/gradlew`.  | required | `$GRADLEW_PATH` |
| `ndk_version` | NDK version to install, for example `23.0.7599858`. Run `sdkmanager --list` on your machine to see all available versions. Leave this input empty if you are not using the Native Development Kit in your project. |  |  |
</details>

<details>
<summary>Outputs</summary>
There are no outputs defined in this step
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-install-missing-android-tools/pulls) and [issues](https://github.com/bitrise-steplib/steps-install-missing-android-tools/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
