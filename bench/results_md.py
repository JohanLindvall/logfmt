#!/usr/bin/env python3
"""Render the cross-library comparison `go test -bench` output (results.txt) into
a markdown summary: one table per scenario, one row per parser, sorted by ns/op,
with a Speedup column relative to go-logfmt (the de-facto standard).

Benchmark names follow `<Scenario>_<Parser>`, e.g. ParseAll_Big_Mine.

Usage: results_md.py <src results.txt> <dst .md>."""
import re
import sys

# Map the benchmark's parser suffix to a display name.
PARSERS = {
    "Mine": "this (logfmt)",
    "GoLogfmt": "go-logfmt",
    "Loki": "Grafana Loki",
    "Kr": "kr/logfmt",
}
BASELINE = "GoLogfmt"

src, dst = sys.argv[1], sys.argv[2]
with open(src) as f:
    lines = f.readlines()

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

# Group by scenario (everything before the trailing _<Parser>), first-seen order.
groups, order = {}, []
for l in lines:
    m = row.match(l.strip())
    if not m:
        continue
    full = m.group(1).lstrip("_")
    scenario, _, parser = full.rpartition("_")
    if parser not in PARSERS:
        scenario, parser = full, ""
    if scenario not in groups:
        groups[scenario] = []
        order.append(scenario)
    groups[scenario].append({
        "parser": parser,
        "ns": float(m.group(2)),
        "mbs": m.group(3),
        "bytes": m.group(4),
        "allocs": m.group(5),
    })

out = ["# logfmt parser comparison", ""]
out += [f"- {m}" for m in meta] + [""]
out.append("This package vs other Go logfmt parsers on the same input. Lower "
           "ns/op is better; throughput (MB/s) and allocations are reported by "
           "`-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.")
out.append("")

for scenario in order:
    rows = groups[scenario]
    base = next((r["ns"] for r in rows if r["parser"] == BASELINE), None)
    out.append(f"## {scenario}")
    out.append("")
    out.append("| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |")
    out.append("|---|--:|--:|--:|--:|--:|")
    for r in sorted(rows, key=lambda r: r["ns"]):
        name = PARSERS.get(r["parser"], r["parser"] or "—")
        mbs = f"{r['mbs']} MB/s" if r["mbs"] else "—"
        b = r["bytes"] if r["bytes"] is not None else "—"
        a = r["allocs"] if r["allocs"] is not None else "—"
        speed = f"{base / r['ns']:.1f}×" if base else "—"
        out.append(f"| {name} | {r['ns']:.0f} | {mbs} | {b} | {a} | {speed} |")
    out.append("")

with open(dst, "w") as f:
    f.write("\n".join(out).rstrip() + "\n")
print(f"wrote {dst}")
