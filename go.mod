module github.com/JohanLindvall/logfmt

// The package uses nothing newer than the builtin clear() (Go 1.21), so the
// floor is kept low deliberately: a library's go directive is the minimum
// toolchain every importer must have, not the version it was developed on.
// CI tests this floor alongside the current stable release.
go 1.21
