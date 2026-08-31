# Audiobook Metadata Plugin for Silo

First-party [Silo](https://github.com/Silo-Server/silo-server) metadata provider
for audiobook libraries. It implements `metadata_provider.v1` with capability ID
`audiobook-metadata` and default priority `audiobook = 2`.

## Sources

The provider searches Audnexus, AudiMeta, iTunes, Audible, Storytel, BookBeat,
Audioteka, and AudiobookCovers. Some sources use public APIs and others parse
their public catalog pages; upstream availability and rate limits still apply.

## Configuration

The plugin has no global configuration. Install it and add **Audiobook
Metadata** to an audiobook library's metadata provider chain.

## Development

```sh
GOWORK=off go test ./...
GOWORK=off make build
```

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. New
sources and matching changes should start as an issue.

## License

`silo-plugin-metadata-audiobook` is licensed under `AGPL-3.0-only`. See
[LICENSE](LICENSE).
