#!/usr/bin/env python3
"""Report what every BSF/BSR in a function waits for besides its source.

AMD and Intel CPUs leave a BSF destination unchanged when the source is zero,
so they treat the destination's previous value as an input. At GOAMD64=v1 Go
compiles bits.TrailingZeros64 to BSF and the register allocator chooses the
destination freely, so a BSF can end up waiting for whatever instruction last
wrote that register. On a Ryzen 8840HS that cost the parser 37% on short
unquoted fields for an instruction stream that was otherwise identical (see
CLAUDE.md, 2026-09-22).

For each BSF this walks the function's control-flow graph backwards and lists
every instruction that can have been the destination's previous writer. A
writer that loads from anywhere but the stack or a global is flagged SUSPECT:
those are the ones that complete late. Run it on a test binary after any
change to Iterate, and investigate any SUSPECT line before timing anything:

    go test -c -o /tmp/logfmt.test .
    python3 bench/bsfdep.py /tmp/logfmt.test 'logfmt\\.Iterate$'
"""

import argparse
import re
import subprocess

REGS = {"AX", "BX", "CX", "DX", "SI", "DI", "R8", "R9", "R10", "R11", "R12", "R13", "R14", "R15"}


def disassemble(binary, func):
    """Return (address, source line, instruction) triples in address order."""
    out = subprocess.run(["go", "tool", "objdump", "-s", func, binary],
                         capture_output=True, text=True, check=True).stdout
    code = []
    for line in out.splitlines():
        fields = [f for f in line.strip().split("\t") if f]
        if len(fields) >= 4 and fields[1].startswith("0x"):
            code.append((int(fields[1], 16), fields[0].rsplit(":", 1)[-1], fields[3].strip()))
    return code


def operands(text):
    m = re.match(r"(\w+)\s*(.*)", text)
    op, args = m.group(1), m.group(2)
    return op, [a.strip() for a in re.split(r",(?![^(]*\))", args)] if args else []


def destination(op, args):
    """The register an instruction writes, in Go assembler operand order."""
    if not args or op.startswith(("CMP", "TEST", "BT", "J", "CALL", "RET", "NOP", "PUSH")):
        return None
    if op.startswith("MOV") and len(args) == 2 and "(" in args[1]:
        return None  # a store
    return args[-1] if args[-1] in REGS else None


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("binary", help="a binary built from this package, e.g. by go test -c")
    ap.add_argument("func", help="go tool objdump -s regexp naming one function")
    args = ap.parse_args()

    code = disassemble(args.binary, args.func)
    index = {addr: k for k, (addr, _, _) in enumerate(code)}
    preds = {k: [] for k in range(len(code))}
    for k, (_, _, text) in enumerate(code):
        op, ops = operands(text)
        target = re.fullmatch(r"0x([0-9a-f]+)", ops[0]) if ops else None
        if op.startswith("J") and target and int(target.group(1), 16) in index:
            preds[index[int(target.group(1), 16)]].append(k)
        if op not in ("JMP", "RET") and k + 1 < len(code):
            preds[k + 1].append(k)

    for k, (addr, line, text) in enumerate(code):
        op, ops = operands(text)
        if op not in ("BSFQ", "BSFL", "BSRQ", "BSRL"):
            continue
        dst = ops[-1]
        if ops[0] == dst:
            print("%x line %-5s %-24s benign: destination is the source" % (addr, line, text))
            continue
        writers, seen, todo = set(), set(), list(preds[k])
        while todo:
            j = todo.pop()
            if j in seen:
                continue
            seen.add(j)
            op2, ops2 = operands(code[j][2])
            if op2.startswith("CALL"):
                writers.add(("call", code[j]))
            elif destination(op2, ops2) == dst:
                src = ops2[0]
                late = op2.startswith("MOV") and "(" in src and "(SP)" not in src and "(SB)" not in src
                writers.add(("LOAD" if late else "ok", code[j]))
            else:
                todo.extend(preds[j])
        verdict = "SUSPECT" if any(kind == "LOAD" for kind, _ in writers) else "benign"
        print("%x line %-5s %-24s %s" % (addr, line, text, verdict))
        for kind, (waddr, wline, wtext) in sorted(writers, key=lambda w: w[1][0]):
            print("    previous write of %s: %x line %-5s %-44s [%s]" % (dst, waddr, wline, wtext, kind))


if __name__ == "__main__":
    main()
