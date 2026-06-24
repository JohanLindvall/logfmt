#!/usr/bin/env python3
"""Render `go test -bench` output from the root logfmt module into a markdown
table (one row per benchmark).

Usage: pkg_results_md.py <src pkg_results.txt> <dst .md>. pkg_bench.sh passes an
architecture-specific destination (pkg_results_<goarch>.md)."""
import re
import sys

src, dst = sys.argv[1], sys.argv[2]
with open(src) as f:
    lines = f.readlines()

# Keep the descriptive '#' facts (generated/go/note); drop the title line.
meta = [l[1:].strip() for l in lines if l.startswith("#") and l[1:].strip()]
meta = [m for m in meta if m.lower().startswith(("generated", "go ", "note"))]

cpu, cores = "", ""
for l in lines:
    if not cpu and l.startswith("cpu:"):
        cpu = l.split(":", 1)[1].strip()
    if not cores:
        m = re.match(r"Benchmark\S+?-(\d+)\s", l)
        if m:
            cores = m.group(1)
if cpu or cores:
    label = "cpu: " + (cpu or "unknown")
    if cores:
        label += f" ({cores} cores)"
    meta.append(label)

row = re.compile(
    r"^Benchmark(\S+?)-\d+\s+\d+\s+([\d.]+)\s*ns/op"
    r"(?:\s+([\d.]+)\s*MB/s)?"
    r"(?:\s+(\d+)\s*B/op)?"
    r"(?:\s+(\d+)\s*allocs/op)?"
)

rows = []
for l in lines:
    m = row.match(l.strip())
    if not m:
        continue
    name = m.group(1).lstrip("_")
    rows.append({
        "name": name,
        "ns": float(m.group(2)),
        "mbs": m.group(3),
        "bytes": m.group(4),
        "allocs": m.group(5),
    })

out = ["# logfmt microbenchmarks", ""]
out += [f"- {m}" for m in meta] + [""]
out.append("The Benchmark* functions in the root logfmt module (parser, lookups, "
           "unescape, ParseTime), as opposed to the cross-library comparison suite "
           "in this `bench/` module (see `results_<arch>.md`). Lower ns/op is "
           "better; throughput (MB/s) and allocations are reported by `-benchmem`.")
out.append("")
out.append("| Benchmark | ns/op | Throughput | B/op | allocs/op |")
out.append("|---|--:|--:|--:|--:|")
for r in rows:
    mbs = f"{r['mbs']} MB/s" if r["mbs"] else "—"
    b = r["bytes"] if r["bytes"] is not None else "—"
    a = r["allocs"] if r["allocs"] is not None else "—"
    out.append(f"| {r['name']} | {r['ns']:.1f} | {mbs} | {b} | {a} |")
out.append("")

with open(dst, "w") as f:
    f.write("\n".join(out).rstrip() + "\n")
print(f"wrote {dst}")
