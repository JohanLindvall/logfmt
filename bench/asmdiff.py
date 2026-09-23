#!/usr/bin/env python3
"""Report which functions of package logfmt compile to different machine code
in two test binaries.

Arch-specific code (scan_arm64.go beside scan_other.go, the constant-selected
decoder in unescape_spare.go) is only safe to change on the architecture that
measured it if every other architecture's machine code stays exactly as it was.
This compares instruction streams function by function, ignoring what moves
when anything else in the binary changes size: addresses, line numbers, jump
targets (made relative to the function), and arm64's page-relative global
addresses. Build the other architecture's binaries with GOARCH set:

    git stash; GOARCH=amd64 go test -c -o /tmp/before.test .; git stash pop
    GOARCH=amd64 go test -c -o /tmp/after.test .
    python3 bench/asmdiff.py /tmp/before.test /tmp/after.test

A function renamed on purpose can be matched with --rename old=new. Test and
benchmark functions differ whenever the tests do; the summary counts them
separately so the package's own functions can be read at a glance.
"""

import argparse
import re
import subprocess
import sys

PKG = "github.com/JohanLindvall/logfmt."


def functions(binary, renames):
    out = subprocess.run(["go", "tool", "objdump", binary], capture_output=True, text=True, check=True).stdout
    funcs, name = {}, None
    for line in out.splitlines():
        if line.startswith("TEXT "):
            name = line.split()[1]
            if not name.startswith(PKG):
                name = None
                continue
            for old, new in renames:
                name = name.replace(old, new)
            funcs[name] = []
            continue
        fields = [f for f in line.strip().split("\t") if f]
        if name is not None and len(fields) >= 4 and fields[1].startswith("0x"):
            funcs[name].append((int(fields[1], 16), fields[3].strip()))
    normal = {}
    for name, code in funcs.items():
        if not code:
            continue
        start, end = code[0][0], code[-1][0]
        body = []
        page = None  # the register the previous instruction loaded a page into
        for _, text in code:
            # amd64 jump targets are absolute; make in-function ones relative.
            text = re.sub(r"0x[0-9a-f]{5,}",
                          lambda m: "@%d" % (int(m.group(0), 16) - start)
                          if start <= int(m.group(0), 16) <= end + 4 else "EXT", text)
            # arm64 reaches a global as ADRP (its page) and then an ADD or a
            # load of the low 12 bits; both numbers move with the layout.
            if page:
                text = re.sub(r"\$\d+, %s, %s" % (page, page), "$LO, %s, %s" % (page, page), text)
                text = re.sub(r"-?\d+\(%s\)" % page, "LO(%s)" % page, text)
            page = None
            m = re.match(r"ADRP -?\d+\(PC\), (R\d+)", text)
            if m:
                page = m.group(1)
                text = "ADRP PAGE, " + page
            for old, new in renames:
                text = text.replace(old, new)
            body.append(text)
        normal[name] = body
    return normal


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("before")
    ap.add_argument("after")
    ap.add_argument("--rename", action="append", default=[], metavar="OLD=NEW",
                    help="treat symbol OLD in either binary as NEW")
    args = ap.parse_args()
    renames = [tuple(r.split("=", 1)) for r in args.rename]
    a, b = functions(args.before, renames), functions(args.after, renames)
    same = {"pkg": 0, "test": 0}
    diff = {"pkg": 0, "test": 0}
    for name in sorted(set(a) | set(b)):
        short = name[len(PKG):]
        kind = "test" if re.match(r"(Test|Benchmark|Fuzz|Example)", short) or "Ref" in short else "pkg"
        if a.get(name) == b.get(name):
            same[kind] += 1
            continue
        diff[kind] += 1
        if name not in a:
            print("only after:  %s" % short)
        elif name not in b:
            print("only before: %s" % short)
        else:
            print("differs:     %s (%d -> %d instructions)" % (short, len(a[name]), len(b[name])))
    print("package functions: %d identical, %d different; test functions: %d identical, %d different"
          % (same["pkg"], diff["pkg"], same["test"], diff["test"]))
    sys.exit(1 if diff["pkg"] else 0)


if __name__ == "__main__":
    main()
