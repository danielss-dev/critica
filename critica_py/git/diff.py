"""Git diff operations."""

import os
import subprocess
from enum import Enum
from pathlib import Path
from typing import Optional


class DiffMode(Enum):
    """Diff mode enumeration."""

    ALL = "all"
    STAGED = "staged"
    UNSTAGED = "unstaged"


def is_git_repository(path: str = ".") -> bool:
    """Check if the given path is within a git repository."""
    try:
        abs_path = Path(path).resolve()

        if not abs_path.exists():
            return False

        # If path is a file, use its directory
        if abs_path.is_file():
            work_dir = abs_path.parent
        else:
            work_dir = abs_path

        result = subprocess.run(
            ["git", "rev-parse", "--git-dir"],
            cwd=work_dir,
            capture_output=True,
            text=True,
        )
        return result.returncode == 0
    except Exception:
        return False


def get_diff(path: str = ".", staged: bool = False) -> str:
    """Get git diff for the specified path.

    Args:
        path: Path to get diff for (default: current directory)
        staged: If True, get staged changes; otherwise get all changes

    Returns:
        Git diff output as string
    """
    mode = DiffMode.STAGED if staged else DiffMode.ALL
    return get_diff_for_mode(path, mode)


def get_diff_for_mode(path: str = ".", mode: DiffMode = DiffMode.ALL) -> str:
    """Get git diff for a specific diff mode.

    Args:
        path: Path to get diff for
        mode: DiffMode enum value

    Returns:
        Git diff output as string
    """
    abs_path = Path(path).resolve()

    # If path is a file, use its directory as working directory
    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    all_diffs = []

    # Get regular diff
    regular_diff = _run_git_diff(str(abs_path), work_dir, mode)
    if regular_diff:
        all_diffs.append(regular_diff)

    # Include untracked files if needed
    if _should_include_untracked(mode):
        untracked_diff = _get_untracked_files_diff(work_dir, abs_path)
        if untracked_diff:
            all_diffs.append(untracked_diff)

    return "\n".join(all_diffs)


def _run_git_diff(abs_path: str, work_dir: Path, mode: DiffMode) -> str:
    """Run git diff command."""
    args = ["git", "diff"]

    if mode == DiffMode.STAGED:
        args.append("--staged")
    elif mode == DiffMode.ALL:
        args.append("HEAD")
    # DiffMode.UNSTAGED uses default (no additional args)

    args.extend(["-U5", "--no-color"])

    if abs_path != ".":
        args.extend(["--", abs_path])

    try:
        result = subprocess.run(
            args, cwd=work_dir, capture_output=True, text=True, check=False
        )

        # Exit code 1 with output is normal for git diff
        if result.returncode == 1 and result.stdout:
            return result.stdout

        if result.returncode != 0:
            if result.stderr.strip():
                raise RuntimeError(f"git diff failed: {result.stderr.strip()}")
            raise RuntimeError(f"git diff failed with exit code {result.returncode}")

        return result.stdout
    except FileNotFoundError:
        raise RuntimeError("git command not found. Please install git.")


def _should_include_untracked(mode: DiffMode) -> bool:
    """Check if untracked files should be included."""
    return mode in (DiffMode.ALL, DiffMode.UNSTAGED)


def _get_untracked_files_diff(work_dir: Path, filter_path: Path) -> str:
    """Get diff for untracked files."""
    try:
        # Get list of untracked files
        result = subprocess.run(
            ["git", "ls-files", "--others", "--exclude-standard"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )

        untracked_files = [f for f in result.stdout.strip().split("\n") if f]
        if not untracked_files:
            return ""

        diffs = []

        for file in untracked_files:
            # Check if file matches filter path
            rel_path = filter_path.relative_to(work_dir) if filter_path != work_dir else Path(".")
            rel_path_str = str(rel_path).replace("\\", "/")

            if rel_path_str != ".":
                # Filter path is a subdirectory or file
                if not (file.startswith(rel_path_str + "/") or file == rel_path_str):
                    continue

            # Read file content
            full_path = work_dir / file
            try:
                content = full_path.read_text(encoding="utf-8", errors="ignore")
            except Exception:
                continue

            # Normalize file path to forward slashes for git diff format
            git_file_path = file.replace("\\", "/")

            # Generate diff format for new file
            lines = content.split("\n")
            diff_lines = [
                f"diff --git a/{git_file_path} b/{git_file_path}",
                "new file mode 100644",
                "index 0000000..0000000",
                "--- /dev/null",
                f"+++ b/{git_file_path}",
                f"@@ -0,0 +1,{len(lines)} @@",
            ]

            for line in lines:
                diff_lines.append(f"+{line}")

            diffs.append("\n".join(diff_lines))

        return "\n".join(diffs)
    except subprocess.CalledProcessError:
        return ""
