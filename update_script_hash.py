#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent
INSTALL_SCRIPTS = (
    ROOT / "install-client.ps1",
    ROOT / "install-client.sh",
    ROOT / "install-server.ps1",
    ROOT / "install-server.sh",
)
COMMIT_QUERY_RE = re.compile(r"(?<=\?commit=)[A-Za-z0-9._-]+")


def run_git(*args: str) -> str:
    return subprocess.check_output(
        ["git", *args],
        cwd=ROOT,
        text=True,
        stderr=subprocess.STDOUT,
    ).strip()


def current_commit_hash() -> str:
    return run_git("rev-parse", "--short", "HEAD")


def update_install_scripts(commit_hash: str) -> list[Path]:
    changed: list[Path] = []

    for script in INSTALL_SCRIPTS:
        original_bytes = script.read_bytes()
        original = original_bytes.decode("utf-8")
        updated, replacements = COMMIT_QUERY_RE.subn(commit_hash, original)

        if replacements == 0:
            raise RuntimeError(f"No ?commit= query parameter found in {script.name}")

        if updated != original:
            script.write_bytes(updated.encode("utf-8"))
            changed.append(script)

    return changed


def install_hook() -> Path:
    hooks_dir = ROOT / ".git" / "hooks"
    if not hooks_dir.is_dir():
        raise RuntimeError("Could not find .git/hooks. Run this from a Git worktree.")

    hook_path = hooks_dir / "pre-commit"
    hook_body = """#!/bin/sh
set -e

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

python_cmd=""
if command -v python >/dev/null 2>&1; then
    python_cmd="python"
elif command -v python3 >/dev/null 2>&1; then
    python_cmd="python3"
else
    echo "pre-commit: python or python3 is required to update install script hashes" >&2
    exit 1
fi

"$python_cmd" update_script_hash.py
git add install-client.ps1 install-client.sh install-server.ps1 install-server.sh
"""
    hook_path.write_text(hook_body, newline="\n")
    try:
        hook_path.chmod(0o755)
    except OSError:
        pass

    return hook_path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Update install script cache-busting commit hashes."
    )
    parser.add_argument(
        "--install-hook",
        action="store_true",
        help="Install a local pre-commit hook that updates and stages install scripts.",
    )
    parser.add_argument(
        "--hash",
        dest="commit_hash",
        help="Commit hash to write. Defaults to the current HEAD short hash.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    if args.install_hook:
        hook_path = install_hook()
        print(f"Installed hook: {hook_path.relative_to(ROOT)}")

    commit_hash = args.commit_hash or current_commit_hash()
    changed = update_install_scripts(commit_hash)

    if changed:
        names = ", ".join(path.name for path in changed)
        print(f"Updated install script hash to {commit_hash}: {names}")
    else:
        print(f"Install script hashes already set to {commit_hash}")

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except subprocess.CalledProcessError as exc:
        print(exc.output, file=sys.stderr, end="")
        raise SystemExit(exc.returncode)
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(1)
