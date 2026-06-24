// Separate module so the root logfmt package stays dependency-free. It compares
// this package against other Go logfmt parsers; run with `go test -bench=.`
// from this directory.
module github.com/JohanLindvall/logfmt/bench

go 1.26.3

require (
	github.com/JohanLindvall/logfmt v0.0.0
	github.com/go-logfmt/logfmt v0.6.1
	github.com/kr/logfmt v0.0.0-20210122060352-19f9bcb100e6
)

replace github.com/JohanLindvall/logfmt => ../
