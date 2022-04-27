# Release Process

For example, to release v0.21.2:

- Update the version numbers in the code: `make release bump=0.21.2`
- Run `make test` and `make lint`, ensure they pass successfully 
- Commit with message `v0.21.2`: `git commit -m "v0.21.2"`
- Tag and push as `v0.21.2`: `git tag v0.21.2 && git push && git push origin v0.21.2`
