# storage-explorer

Load a payloads file and browse its accounts, their storage maps, and their stored values.

## Usage

```shell
npm i
npm run build
go run . -port 4000 -payloads payloads-file
```

## Development

```shell
gow -e go,gohtml run . -port 4000 -payloads payloads-file
```

and

```shell
npx vite build --watch
```
