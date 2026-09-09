#!/usr/bin/env python3
"""Compare prebuilt Go benchmark binaries in alternating, CPU-pinned rounds.

Build both binaries with the same tests and toolchain. Run an A/A control with
the same binary in both positions before comparing a changed implementation.
Keep builds, fuzzers and other benchmarks out of the measurement interval.
"""

import argparse
import pathlib
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("before", type=pathlib.Path)
    parser.add_argument("after", type=pathlib.Path)
    parser.add_argument("--cpu", type=int, required=True, help="Linux CPU to pin both binaries to")
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--bench", default=".", help="Go benchmark regular expression")
    parser.add_argument("--rounds", type=int, default=6)
    parser.add_argument("--benchtime", default="1s")
    parser.add_argument("--benchstat", default="benchstat")
    parser.add_argument("--cwd", type=pathlib.Path, default=pathlib.Path.cwd(),
                        help="Module directory containing the benchmark's testdata")
    args = parser.parse_args()
    if args.rounds < 1:
        parser.error("--rounds must be positive")
    binaries = [args.before.resolve(strict=True), args.after.resolve(strict=True)]
    args.output.mkdir(parents=True, exist_ok=True)
    paths = [args.output / "before.txt", args.output / "after.txt"]
    with paths[0].open("w") as before, paths[1].open("w") as after:
        files = [before, after]
        for round_no in range(args.rounds):
            for index in ([0, 1] if round_no % 2 == 0 else [1, 0]):
                subprocess.run([
                    "taskset", "-c", str(args.cpu), str(binaries[index]),
                    "-test.run=^$", "-test.bench=" + args.bench,
                    "-test.benchmem", "-test.benchtime=" + args.benchtime,
                    "-test.count=1",
                ], stdout=files[index], cwd=args.cwd, check=True)
                files[index].flush()
            print(f"Round {round_no + 1}/{args.rounds} complete", flush=True)
    subprocess.run([args.benchstat, *map(str, paths)], check=True)


if __name__ == "__main__":
    main()
