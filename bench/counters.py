#!/usr/bin/env python3
"""Compare per-op hardware counters of prebuilt Go benchmark binaries.

Cycles are core cycles, so unlike ns/op they do not move with the clock:
frequency and power-state drift, which moves this repo's timings by up to 30%
between sessions, leaves them alone. Macro-op, instruction and branch counts
are exact. That makes this a fast first screen for a change -- a few seconds
per benchmark -- before the alternating, pinned timing series of compare.py.

Each count is taken by the difference method: the benchmark runs twice, for N1
and N2 iterations, and the result is (count(N2) - count(N1)) / (N2 - N1), which
cancels process start-up, init and the testing package's N=1 probe run. The
binaries are interleaved, the whole measurement is repeated --reps times, and
the run with the fewest cycles is kept as the least disturbed one.

Needs Linux perf with access to the core PMU (perf_event_paranoid <= 2 counts
the benchmark's own user-mode events). The event set follows the machine: AMD
Zen's retired macro-ops and branches on x86-64, and on arm64 the Armv8 PMUv3
common events by number -- OP_RETIRED, BR_RETIRED, STALL_BACKEND and
STALL_FRONTEND, which the Neoverse N2 VMs expose (the IMPDEF and STALL_SLOT
events read 0 there). The two stall columns are the share of cycles in which
nothing was dispatched for want of backend or frontend resources. Build both
binaries with the same test sources and toolchain; pass benchmark names without
the -N CPU suffix:

    python3 bench/counters.py 'Benchmark_IterateOur,Benchmark_Get/level' \\
        /tmp/before.test /tmp/after.test --cpu 10
"""

import argparse
import os
import platform
import subprocess
import sys

# Each set is cycles, ops, instructions, branches, then extras; one group, so
# every event is counted over the same interval (7 counters on the N2).
EVENTS = {
    "amd": ["cycles:u", "ex_ret_ops:u", "instructions:u", "ex_ret_brn:u", "ex_ret_brn_misp:u"],
    # OP_RETIRED, BR_RETIRED, STALL_BACKEND, STALL_FRONTEND, BR_MIS_PRED_RETIRED
    "arm": ["cycles:u", "r003a:u", "instructions:u", "r0021:u", "r0024:u", "r0023:u", "r0022:u"],
    "portable": ["cycles:u", "instructions:u", "instructions:u", "branches:u", "branch-misses:u"],
}


def measure(binary, bench, n, events, cpu, cwd):
    # -test.bench splits on '/' and matches each part on its own, so every part
    # is anchored: an unanchored "quoted" also runs "unquoted".
    pattern = "/".join("^%s$" % part for part in bench.split("/"))
    p = subprocess.run(["perf", "stat", "-x,", "-e", "{" + ",".join(events) + "}",
                        "taskset", "-c", str(cpu), binary, "-test.run=^$",
                        "-test.bench=" + pattern, "-test.benchtime=%dx" % n, "-test.count=1"],
                       capture_output=True, text=True, cwd=cwd)
    if "PASS" not in p.stdout:
        sys.exit("%s %s failed:\n%s%s" % (binary, bench, p.stdout, p.stderr))
    counts = {}
    for line in p.stderr.splitlines():
        f = line.split(",")
        if len(f) >= 3 and f[2] in events:
            if not f[0][:1].isdigit():
                sys.exit("perf could not count %s: %s" % (f[2], line))
            counts[f[2]] = int(f[0])
    return counts


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("benches", help="comma-separated benchmark names")
    ap.add_argument("binaries", nargs="+")
    ap.add_argument("--cpu", type=int, required=True, help="Linux CPU to pin every run to")
    ap.add_argument("--reps", type=int, default=3)
    ap.add_argument("--seconds", type=float, default=0.25, help="approximate length of the longer run")
    ap.add_argument("--cwd", default=os.getcwd(), help="module directory holding the benchmark's testdata")
    ap.add_argument("--portable", action="store_true",
                    help="count generic events only, for CPUs without the AMD Zen or Arm ones")
    args = ap.parse_args()
    pmu = "portable" if args.portable else "arm" if platform.machine() in ("aarch64", "arm64") else "amd"
    events = list(dict.fromkeys(EVENTS[pmu]))
    cyc, ops, ins, brn = EVENTS[pmu][:4]
    names = [os.path.basename(b) for b in args.binaries]
    stalls = "  %6s %6s" % ("be%", "fe%") if pmu == "arm" else ""
    stalls += "  %7s" % "misp/op" if pmu != "portable" else ""
    print("%-44s %-16s %10s %10s %10s %9s%s" % ("benchmark", "binary", "cycles/op", "ops/op", "instr/op", "brn/op", stalls))
    for bench in args.benches.split(","):
        probe = measure(args.binaries[0], bench, 1000, events, args.cpu, args.cwd)
        per_op = max(probe[cyc] / 1000.0, 1.0)
        n2 = max(int(args.seconds * 4e9 / per_op), 600)
        n1 = n2 // 6
        best = {}
        for rep in range(args.reps):
            order = range(len(args.binaries)) if rep % 2 == 0 else reversed(range(len(args.binaries)))
            for i in order:
                c1 = measure(args.binaries[i], bench, n1, events, args.cpu, args.cwd)
                c2 = measure(args.binaries[i], bench, n2, events, args.cpu, args.cwd)
                r = {e: (c2[e] - c1[e]) / float(n2 - n1) for e in events}
                if i not in best or r[cyc] < best[i][cyc]:
                    best[i] = r
        for i, name in enumerate(names):
            r = best[i]
            extra = ""
            if pmu == "arm":
                extra = "  %6.1f %6.1f" % (100 * r["r0024:u"] / r[cyc], 100 * r["r0023:u"] / r[cyc])
            if pmu != "portable":
                extra += "  %7.2f" % r[EVENTS[pmu][-1]]
            if i:
                extra += "  cycles %+6.1f%%" % (100 * (r[cyc] / best[0][cyc] - 1))
            print("%-44s %-16s %10.1f %10.1f %10.1f %9.1f%s" % (
                bench if i == 0 else "", name, r[cyc], r[ops] if pmu != "portable" else float("nan"),
                r[ins], r[brn], extra))
        sys.stdout.flush()


if __name__ == "__main__":
    main()
