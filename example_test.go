// SPDX-License-Identifier: MIT

package logfmt_test

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/JohanLindvall/logfmt"
)

// The README's quick start: keep its main function and this example in step.
// pkg.go.dev shows this example with a Run button.
func Example() {
	line := []byte(`level=error msg="disk \"sda\" full" retries=3`)

	// One key, zero-copy: level is a sub-slice of line.
	level, _ := logfmt.Get(line, "level")
	fmt.Printf("level: %s\n", level)

	// Escapes decoded into a buffer you own.
	msg, _ := logfmt.AppendValue(nil, line, "msg")
	fmt.Printf("msg: %s\n", msg)

	// Every pair in order, values raw; return false to stop early.
	err := logfmt.Iterate(line, func(key, val []byte) bool {
		fmt.Printf("%s=%s\n", key, val)
		return true
	})
	if err != nil {
		fmt.Println(err)
	}

	// Output:
	// level: error
	// msg: disk "sda" full
	// level=error
	// msg=disk \"sda\" full
	// retries=3
}

func ExampleIterate() {
	line := []byte(`level=info msg="user login" user=john debug`)

	err := logfmt.Iterate(line, func(key, val []byte) bool {
		if logfmt.IsBareKey(val) {
			fmt.Printf("%s (bare key)\n", key)
			return true
		}
		fmt.Printf("%s = %s\n", key, val)
		return true
	})
	fmt.Println("err:", err)

	// Output:
	// level = info
	// msg = user login
	// user = john
	// debug (bare key)
	// err: <nil>
}

func ExampleGet() {
	line := []byte(`level=warn msg= user=john`)

	level, ok := logfmt.Get(line, "level")
	fmt.Printf("level: %q, found=%v\n", level, ok)

	// Present with an empty value: a non-nil, zero-length slice.
	msg, ok := logfmt.Get(line, "msg")
	fmt.Printf("msg: %q, nil=%v, found=%v\n", msg, msg == nil, ok)

	// Absent.
	id, ok := logfmt.Get(line, "id")
	fmt.Printf("id: %q, nil=%v, found=%v\n", id, id == nil, ok)

	// Output:
	// level: "warn", found=true
	// msg: "", nil=false, found=true
	// id: "", nil=true, found=false
}

func ExampleGetQuoted() {
	line := []byte(`msg="disk \"sda\" full" path=C:\tmp`)
	var buf []byte

	for _, key := range []string{"msg", "path"} {
		v, quoted, ok := logfmt.GetQuoted(line, key)
		if !ok {
			continue
		}
		// Escapes mean something only inside quotes: path's \t is two
		// literal bytes, not a tab.
		if quoted && logfmt.NeedsUnescape(v) {
			buf = logfmt.AppendUnescape(buf[:0], v)
			v = buf
		}
		fmt.Printf("%s: %s (quoted=%v)\n", key, v, quoted)
	}

	// Output:
	// msg: disk "sda" full (quoted=true)
	// path: C:\tmp (quoted=false)
}

func ExampleGetMany() {
	line := []byte(`ts=2026-10-06T21:00:00Z level=info msg="request done" status=200`)
	keys := []string{"level", "status", "trace_id"}

	var buf [][]byte // pass the previous result back in to reuse it
	buf = logfmt.GetMany(line, keys, buf)
	for i, v := range buf {
		if v == nil {
			fmt.Printf("%s: absent\n", keys[i])
			continue
		}
		fmt.Printf("%s: %s\n", keys[i], v)
	}

	// Output:
	// level: info
	// status: 200
	// trace_id: absent
}

func ExampleGetMany_reuseBuffer() {
	lines := [][]byte{
		[]byte(`level=info status=200`),
		[]byte(`level=error msg="request failed"`),
	}
	keys := []string{"level", "status"}
	buf := make([][]byte, len(keys))

	for _, line := range lines {
		// Keep the returned slice so the next call can reuse its storage.
		buf = logfmt.GetMany(line, keys, buf)
		for i, v := range buf {
			if v == nil {
				fmt.Printf("%s: absent\n", keys[i])
				continue
			}
			fmt.Printf("%s: %s\n", keys[i], v)
		}
	}

	// Output:
	// level: info
	// status: 200
	// level: error
	// status: absent
}

func ExampleAppendValue() {
	line := []byte(`msg="first\nsecond" path=C:\Users\bob\new`)

	// A quoted value has its escapes decoded...
	msg, _ := logfmt.AppendValue(nil, line, "msg")
	fmt.Printf("%s\n", msg)

	// ...an unquoted one is copied byte for byte.
	path, _ := logfmt.AppendValue(nil, line, "path")
	fmt.Printf("%s\n", path)

	// Output:
	// first
	// second
	// C:\Users\bob\new
}

func ExampleAppendValue_reuseBuffer() {
	lines := [][]byte{
		[]byte(`msg="first\nsecond"`),
		[]byte(`msg=done`),
	}
	buf := make([]byte, 0, 64)

	for _, line := range lines {
		var ok bool
		// Reset the length, then keep the returned slice for the next call.
		buf, ok = logfmt.AppendValue(buf[:0], line, "msg")
		if ok {
			fmt.Printf("%q\n", buf)
		}
	}

	// Output:
	// "first\nsecond"
	// "done"
}

func Example_readStream() {
	input := strings.NewReader("level=info msg=started\nlevel=error msg=failed\n")
	scanner := bufio.NewScanner(input)
	// Choose a maximum record size for your application; Scanner's default
	// is 64 KiB. Allow up to 1 MiB here, including the line ending.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		level, _ := logfmt.Get(scanner.Bytes(), "level")
		// Use the value before Scan reuses its buffer; copy it to retain it.
		fmt.Printf("%s\n", level)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("read logs:", err)
	}

	// Output:
	// info
	// error
}

func ExampleSplitRecord() {
	data := []byte("level=info msg=started\r\nlevel=error msg=failed\n")

	for len(data) > 0 {
		var rec []byte
		rec, data = logfmt.SplitRecord(data)
		level, _ := logfmt.Get(rec, "level")
		msg, _ := logfmt.Get(rec, "msg")
		fmt.Printf("%s: %s\n", level, msg)
	}

	// Output:
	// info: started
	// error: failed
}

func ExampleValidate() {
	line := []byte(`level=info msg="unterminated`)

	err := logfmt.Validate(line)
	var se *logfmt.SyntaxError
	if errors.As(err, &se) {
		fmt.Printf("offset %d: %s\n", se.Offset, se.Reason)
	}
	fmt.Println("ErrBadFormat:", errors.Is(err, logfmt.ErrBadFormat))

	// The lookups stop once their key is settled, so they never reach the fault.
	level, ok := logfmt.Get(line, "level")
	fmt.Printf("level: %s, found=%v\n", level, ok)

	// Output:
	// offset 15: unterminated quoted value
	// ErrBadFormat: true
	// level: info, found=true
}

func ExampleParseTime() {
	for _, ts := range []string{
		"2026-10-06T21:00:00.123456789+02:00",
		"1748239806.3691056",
		"1748239806369", // a millisecond epoch: rejected, not guessed at
	} {
		t, ok := logfmt.ParseTime([]byte(ts))
		fmt.Println(t, ok)
	}

	// Output:
	// 2026-10-06 19:00:00.123456789 +0000 UTC true
	// 2025-05-26 06:10:06.3691056 +0000 UTC true
	// 0001-01-01 00:00:00 +0000 UTC false
}
