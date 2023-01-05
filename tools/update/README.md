# Update

The update tool helps with updating Flow downstream dependencies to a new Cadence version.
It is able to automate tedious tasks like bumping versions, opening PRs, tagging, etc.
The tool automatically detects versions and supports multiple modules per repo.

## Usage

- Install dependencies:

  ```sh
  npm i -g typescript ts-node
  ```

- Run:

  ```sh
  GITHUB_TOKEN=`gh auth token` ts-node main.ts update --version <version>
  ```

Certain dependencies are not released, e.g. flow-go.
Once updated, provide their version manually, e.g.

  ```sh
  GITHUB_TOKEN=`gh auth token` ts-node main.ts update --version <version> --versions onflow/flow-go@<commit>
  ```

Dependencies are configured in `config.yaml`.

## Development

- When updating the JSON Schema, regenerate the TypeScript definition:

  ```sh
  ./node_modules/json-schema-to-typescript/dist/src/cli.js config.schema.json > config.schema.ts
  ```
