# Update Tool

The update tool helps with updating Flow downstream dependencies to a new Cadence version.
It is able to automate tedious tasks like bumping versions, opening PRs, tagging, etc.
The tool automatically detects versions and supports multiple modules per repo.

## Usage

### Install Dependencies

```sh
npm i -g typescript ts-node
```

### Run

```sh
GH_TOKEN=<github_token> ts-node main.ts update --version <version>
```

If the github CLI (`gh`) is installed and configured, the auth token can also be retrieved from the github CLI,
instead of manually providing.

```sh
GH_TOKEN=`gh auth token` ts-node main.ts update --version <version>
```

The `update` command use HTTPS to connect to github by default. To use SSH instead, use `--useSSH true` as arguments in
the command.

By default, the tool does not release certain dependencies. e.g. flow-go.
These repos have to be manually released/tagged.

#### Using existing versions

Suppose there are already released versions of certain downstream dependencies.
For example suppose we want to update downstream dependencies to Cadence `v0.30.0`,
and the `flow-go-sdk` is already updated with that Cadence version and is already tagged as `v0.31.0`.
Similarly, suppose `flow-go` is also already updated with that Cadence version and is already tagged as `v0.26.0`.

Then, updating the remaining of downstream dependencies can be done by providing the already released versions manually
to the update command. The `--versions` flag would take a comma separated set of repos.

```sh
GH_TOKEN=<github_token> ts-node main.ts update --version v0.30.0 --versions onflow/flow-go-sdk@v0.31.0,onflow/flow-go@v0.26.0
```

Above will update the rest of the dependencies to:

- Cadence `v0.30.0`
- flow-go-sdk `v0.31.0`
- flow-go `v0.26.0`

Instead of the version, it is also possible to provide a commit,
in Go's expected format, i.e. the first 12 characters of the commit hash.

```sh
GH_TOKEN=`gh auth token` ts-node main.ts update --version v0.30.0 --versions onflow/flow-go@<commit>
```

#### Configuring dependencies

Dependencies are configured in `config.yaml`.

### Process

The update process is basically a cycle of the following steps for each downstream dependency:

1. Use the tool to open a PR that updates the dependencies
2. Review the PR for correctness
3. Get the PR reviewed, approved, and merged
4. Release a new version of the downstream dependency
5. Go to 1

### Example

Let's walk through an example scenario.

Cadence release `v1.0.0-M8` was just published.

#### Updating the first downstream dependency

Start the process of updating all downstream dependencies by invoking the `update` subcommand with the new Cadence version.

The Cadence version needs to be provided using the `--version` flag.

```shell
$ GH_TOKEN=`gh auth token` ts-node main.ts update \
    --version v1.0.0-M8
```

<details>
<summary>
Output:
</summary>

```
👋 Hello, turbolent
Updating all repos to onflow/cadence version v1.0.0-M8

Checking repo onflow/flow-go-sdk ...
  > Latest release of repo onflow/flow-go-sdk: v0.41.20
  > Checking if all mods of repo onflow/flow-go-sdk at version v0.41.20 are updated ...
  > Checking if mod onflow/flow-go-sdk at repo version v0.41.20 is updated ...
  > Outdated dep onflow/cadence: expected v1.0.0-M8, got v0.42.6
  > Latest release of repo onflow/flow-go-sdk (v0.41.20) is not updated, checking default branch ...
  > Default branch of repo onflow/flow-go-sdk: master
  > Checking if all mods of repo onflow/flow-go-sdk at version 8bf96750a5e3c057cfe8fad52865c5fa9afd0fba are updated ...
  > Checking if mod onflow/flow-go-sdk at repo version 8bf96750a5e3c057cfe8fad52865c5fa9afd0fba is updated ...
  > Outdated dep onflow/cadence: expected v1.0.0-M8, got M7
  > Checking if an update PR exists ...
    > Checking if PR 583 updates a dep of a mod ...
    > Checking if PR 582 updates a dep of a mod ...
    > Checking if PR 581 updates a dep of a mod ...
    > Checking if PR 580 updates a dep of a mod ...
    > Checking if PR 578 updates a dep of a mod ...
    > Checking if PR 574 updates a dep of a mod ...
    > Checking if PR 572 updates a dep of a mod ...
    > Checking if PR 565 updates a dep of a mod ...
    > Checking if PR 542 updates a dep of a mod ...
    > Checking if PR 508 updates a dep of a mod ...
    > Checking if PR 487 updates a dep of a mod ...
    > Checking if PR 480 updates a dep of a mod ...
    > Checking if PR 290 updates a dep of a mod ...
  > No update PR found
✔ Would you like to update repo 'onflow/flow-go-sdk' and create a PR? … yes
  Cloning onflow/flow-go-sdk ...
Cloning into '/var/folders/n6/04ql0mr94nq5qj61wz_lcsx40000gn/T/onflow-flow-go-sdkmD43jW'...
  Creating branch auto-update-onflow-cadence-v1.0.0-M8 ...
Switched to a new branch 'auto-update-onflow-cadence-v1.0.0-M8'
  Updating mod onflow/flow-go-sdk ...
  Updating mod onflow/flow-go-sdk to github.com/onflow/cadence@v1.0.0-M8 ...
go: downloading github.com/onflow/cadence v1.0.0-M8
go: upgraded github.com/onflow/cadence v1.0.0-M7 => v1.0.0-M8
  Cleaning up mod onflow/flow-go-sdk ...
  Committing update ...
[auto-update-onflow-cadence-v1.0.0-M8 3a172b5] Update to Cadence v1.0.0-M8
 2 files changed, 3 insertions(+), 3 deletions(-)
  Pushing update ...
remote:
remote: Create a pull request for 'auto-update-onflow-cadence-v1.0.0-M8' on GitHub by visiting:
remote:      https://github.com/onflow/flow-go-sdk/pull/new/auto-update-onflow-cadence-v1.0.0-M8
remote:
remote: GitHub found 2 vulnerabilities on onflow/flow-go-sdk's default branch (2 moderate). To find out more, visit:
remote:      https://github.com/onflow/flow-go-sdk/security/dependabot
remote:
To ssh://github.com/onflow/flow-go-sdk
 * [new branch]      auto-update-onflow-cadence-v1.0.0-M8 -> auto-update-onflow-cadence-v1.0.0-M8
branch 'auto-update-onflow-cadence-v1.0.0-M8' set up to track 'origin/auto-update-onflow-cadence-v1.0.0-M8'.
  Creating PR ...
  Created PR https://github.com/onflow/flow-go-sdk/pull/584
  Cleaning up clone of onflow/flow-go-sdk ...
```

</details>

The command will determine that the `flow-go-sdk` downstream dependency is outdated and will propose to open a PR for it.

Agree to have a PR opened, get it reviewed, approved, and merged.

Check the GitHub releases page and determine the next version, e.g. for `flow-go-sdk`, https://github.com/onflow/flow-go-sdk/releases.

In this example, the next version is `v1.0.0-M5`.

Use the `release` subcommand to create a new tag:

```shell
$ GH_TOKEN=`gh auth token` ts-node main.ts release --repo onflow/flow-go-sdk --version v1.0.0-M5
```

<details>
<summary>
Output:
</summary>

```
👋 Hello, turbolent
Cloning onflow/flow-go-sdk ...
Cloning into '/var/folders/n6/04ql0mr94nq5qj61wz_lcsx40000gn/T/onflow-flow-go-sdk1Ptv5n'...
Tagging onflow/flow-go-sdk version v1.0.0-M5 ...
Pushing onflow/flow-go-sdk version v1.0.0-M5 ...
To ssh://github.com/onflow/flow-go-sdk
 * [new tag]         v1.0.0-M5 -> v1.0.0-M5
Cleaning up clone of onflow/flow-go-sdk
Now create a GitHub release: https://github.com/onflow/flow-go-sdk/releases/new?tag=v1.0.0-M5
```

</details>

Go to the link that is shown at the end to create a new GitHub (pre-)release for this dependency, e.g. https://github.com/onflow/flow-go-sdk/releases/new?tag=v1.0.0-M5.

> [!IMPORTANT]
> Cadence 1.0 is not released yet, so all releases MUST be PRE-RELEASES!

#### Updating the next downstream dependency

Once the GitHub release has been published, re-run the `update` subcommand. The tool will determine that the next dependency needs to be updated.

Versions are specified using the `--versions` flag, comma-separated.

```shell
GH_TOKEN=`gh auth token` ts-node main.ts update \
    --version v1.0.0-M8
```

Again, the tool will determine the next downstream dependency that needs to get updated.
This time is the `lint` module in the `onflow/cadence-tools` repo.

<details>
<summary>
Output:
</summary>

```
👋 Hello, turbolent
Updating all repos to onflow/cadence version v1.0.0-M7

Checking repo onflow/flow-go-sdk ...
  > Checking if all mods of repo onflow/flow-go-sdk at version v1.0.0-M5 are updated ...
  ✓ All mods of mod onflow/flow-go-sdk at repo version v1.0.0-M5 are up-to-date

Checking repo onflow/cadence-tools ...
  > Latest release of repo onflow/cadence-tools: languageserver/v0.33.4
  > Checking if all mods of repo onflow/cadence-tools at version languageserver/v0.33.4 are updated ...
  > Checking if mod onflow/cadence-tools/lint at repo version languageserver/v0.33.4 is updated ...
  > Outdated dep onflow/cadence: expected v1.0.0-M7, got v0.42.5
  > Latest release of repo onflow/cadence-tools (languageserver/v0.33.4) is not updated, checking default branch ...
  > Default branch of repo onflow/cadence-tools: master
  > Checking if all mods of repo onflow/cadence-tools at version afa07708b24252156efdc9c9f1ae62b69d2c0d6a are updated ...
  > Checking if mod onflow/cadence-tools/lint at repo version afa07708b24252156efdc9c9f1ae62b69d2c0d6a is updated ...
  > Outdated dep onflow/flow-go-sdk: expected v1.0.0-M5, got M4
  > Checking if an update PR exists ...
    > Checking if PR 297 updates a dep of a mod ...
    > Checking if PR 297 updates dep onflow/cadence to v1.0.0-M7 ...
    > Checking if PR 297 updates dep onflow/flow-go-sdk to v1.0.0-M5 ...
    > PR https://github.com/onflow/cadence-tools/pull/297 is not an update PR
    > Checking if PR 292 updates a dep of a mod ...
    > Checking if PR 292 updates dep onflow/cadence to v1.0.0-M7 ...
    > Checking if PR 292 updates dep onflow/flow-go-sdk to v1.0.0-M5 ...
    > PR https://github.com/onflow/cadence-tools/pull/292 is not an update PR
    > Checking if PR 286 updates a dep of a mod ...
    > Checking if PR 279 updates a dep of a mod ...
    > Checking if PR 279 updates dep onflow/cadence to v1.0.0-M7 ...
    > Checking if PR 279 updates dep onflow/flow-go-sdk to v1.0.0-M5 ...
    > PR https://github.com/onflow/cadence-tools/pull/279 is not an update PR
    > Checking if PR 275 updates a dep of a mod ...
    > Checking if PR 275 updates dep onflow/cadence to v1.0.0-M7 ...
    > Checking if PR 275 updates dep onflow/flow-go-sdk to v1.0.0-M5 ...
    > PR https://github.com/onflow/cadence-tools/pull/275 is not an update PR
    > Checking if PR 271 updates a dep of a mod ...
    > Checking if PR 176 updates a dep of a mod ...
  > No update PR found
✖ Would you like to update repo 'onflow/cadence-tools' and create a PR? …
```

</details>

#### Caveats to be aware of

> [!IMPORTANT]
> For some downstream dependencies,

- Some downstream dependencies, like `flow-go`, are not tagged/released.

  Use the latest commit instead, in Go's expected format, i.e. the first 12 characters of the commit hash.

  For example, to `flow-emulator` depends on `flow-go`, and it can be updated using:

  ```shell
  $ GH_TOKEN=`gh auth token` ts-node main.ts update \
      --version v1.0.0-M8 \
      --versions onflow/flow-go@3677206d445c
  ```

- Some downstream dependencies are modules in the same repo.

  When releasing, specify the `--mod` flag.

  For example, to release module `lint` in repo `cadence-tools`:

  ```shell
  $ GH_TOKEN=`gh auth token` ts-node main.ts release \
      -r onflow/cadence-tools \
      --mod lint \
      --version v1.0.0-M5
  ```

- `flowkit` is at `v2`, which Go handles specially, and the update tool does not quite support out-of-the-box yet.

  Specify the version both for `onflow/flowkit` and for `onflow/flowkit/v2`.

  For example, to update a downstream dependency of `flowkit` to `v2.0.0-stable-cadence-alpha.5`, use `onflow/flowkit@v2.0.0-stable-cadence-alpha.5,onflow/flowkit/v2@v2.0.0-stable-cadence-alpha.5`.

## Development

When updating the JSON Schema, regenerate the TypeScript definition:

```sh
./node_modules/json-schema-to-typescript/dist/src/cli.js config.schema.json > config.schema.ts
```
