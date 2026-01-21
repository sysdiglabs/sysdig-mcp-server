# Releasing

The workflow in .github/workflows/publish.yaml will create a new release automatically when the version of the crate changes in package.nix in the default git branch.
So, if you attempt to release a new version, you need to update this version. You should try releasing a new version when you do any meaningful change that the user can benefit from.
The guidelines to follow would be:

* New feature is implemented -> Release new version.
* Bug fixes -> Release new version.
* CI/Refactorings/Internal changes -> No need to release new version.
* Documentation changes -> No need to release new version.

The project follows the [Semver spec](https://semver.org/spec/v2.0.0.html) with these guidelines:

* **MAJOR (X.0.0)**: Breaking changes that are not backward compatible.
* **MINOR (1.X.0)**: New functionality that is backward compatible.
* **PATCH (1.0.X)**: Bug fixes that are backward compatible.
* Before choosing the version bump, check all the commits since the last tag.

After the commit is merged into the default branch the workflow will cross-compile the project and create a GitHub release of that version.
Check the workflow file in case of doubt.
