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

Instead of the version, it is also possible to provide a commit id.

```sh
GH_TOKEN=`gh auth token` ts-node main.ts update --version v0.30.0 --versions onflow/flow-go@<commit_id>
```

#### Configuring dependencies

Dependencies are configured in `config.yaml`.

## Development

When updating the JSON Schema, regenerate the TypeScript definition:

```sh
./node_modules/json-schema-to-typescript/dist/src/cli.js config.schema.json > config.schema.ts
```
