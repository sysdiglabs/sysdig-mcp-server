# Releasing

The workflow in .github/workflows/publish.yaml will create a new release automatically when the version of the crate changes in package.nix in the default git branch.
So, if you attempt to release a new version, you need to update this version. You should try releasing a new version when you do any meaningful change that the user can benefit from.
The guidelines to follow would be:

* New feature is implemented -> Release new version.
* Bug fixes -> Release new version.
* CI/Refactorings/Internal changes -> No need to release new version.
* Documentation changes -> No need to release new version.

The current version of the project is not stable yet, so you need to follow the [Semver spec](https://semver.org/spec/v2.0.0.html), with the following guidelines:

* Unless specified, do not attempt to stabilize the version. That is, do not try to update the version to >=1.0.0. Versions for now should be <1.0.0.
* For minor changes, update only the Y in 0.X.Y. For example: 0.5.2 -> 0.5.3
* For major/feature changes, update the X in 0.X.Y and set the Y to 0. For example: 0.5.2 -> 0.6.0
* Before choosing if the changes are minor or major, check all the commits since the last tag.

After the commit is merged into the default branch the workflow will cross-compile the project and create a GitHub release of that version.
Check the workflow file in case of doubt.
