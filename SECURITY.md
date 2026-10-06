# Security policy

## Reporting a vulnerability

Report security issues privately through GitHub's private vulnerability
reporting: open this repository's **Security** tab and choose **Report a
vulnerability**, or go straight to
<https://github.com/JohanLindvall/logfmt/security/advisories/new>. Please do not
open a public issue or pull request for a suspected vulnerability.

A useful report includes:

- the input that triggers it — a failing test, or the corpus file that
  `go test -fuzz` wrote, is ideal;
- the function you called;
- the Go version and `GOARCH` it reproduces on. arm64 runs partly different
  code from the other architectures (`scan_arm64.go` against `scan_other.go`,
  and `unescape_spare.go`), so a fault may show on one of them only.

## Scope

This package parses log data, which an attacker can often influence. On any
input, however malformed, treat these as security issues:

- a panic, a hang, or any other crash;
- running time or memory that grows faster than linearly with the length of
  the data;
- reading or writing memory outside the slices you passed in.

A wrong result that is not a crash — a value decoded incorrectly, a key
missed — is an ordinary bug: open a public issue, unless you believe it is
exploitable, in which case report it privately as above.

## Supported versions

Only the latest version is supported. The module is pre-1.0, and every green
build of `main` is tagged as the next patch version, so a fix ships as a new
`v0.0.x` tag:

```sh
go get github.com/JohanLindvall/logfmt@latest
```
