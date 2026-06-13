---
title: "Installation"
description: "Install s2 from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/semanticscholar-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `s2` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/semanticscholar-cli/cmd/s2@latest
```

That puts `s2` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/semanticscholar-cli
cd semanticscholar-cli
make build        # produces ./bin/s2
./bin/s2 version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/s2:latest --help
```

## Checking the install

```bash
s2 version
```

prints the version and exits.
