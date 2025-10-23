"""Git branch operations."""

import subprocess
from pathlib import Path
from typing import List


def get_current_branch(path: str = ".") -> str:
    """Get the current branch name.

    Args:
        path: Repository path (default: current directory)

    Returns:
        Current branch name
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        result = subprocess.run(
            ["git", "branch", "--show-current"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to get current branch: {e}")


def get_all_branches(path: str = ".") -> List[str]:
    """Get list of all branches (local and remote).

    Args:
        path: Repository path (default: current directory)

    Returns:
        List of branch names
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        result = subprocess.run(
            ["git", "branch", "-a"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )

        branches = set()
        for line in result.stdout.strip().split("\n"):
            line = line.strip()
            if not line:
                continue

            # Remove * prefix for current branch
            line = line.removeprefix("* ")
            # Remove remotes/ prefix
            line = line.removeprefix("remotes/")

            # Skip HEAD references
            if "HEAD" in line:
                continue

            # Remove origin/ prefix for cleaner display
            line = line.removeprefix("origin/")

            branches.add(line)

        return sorted(list(branches))
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to get branches: {e}")


def get_remote_branches(path: str = ".") -> List[str]:
    """Get list of remote branches.

    Args:
        path: Repository path (default: current directory)

    Returns:
        List of remote branch names
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        result = subprocess.run(
            ["git", "branch", "-r"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )

        branches = set()
        for line in result.stdout.strip().split("\n"):
            line = line.strip()
            if not line:
                continue

            # Skip HEAD references
            if "HEAD" in line:
                continue

            # Remove remotes/ prefix
            line = line.removeprefix("remotes/")

            branches.add(line)

        return sorted(list(branches))
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to get remote branches: {e}")


def get_branch_diff(path: str, from_branch: str, to_branch: str) -> str:
    """Get diff between two branches.

    Args:
        path: Repository path
        from_branch: Source branch name
        to_branch: Target branch name (usually main/master)

    Returns:
        Git diff output as string
    """
    if not from_branch:
        raise ValueError("Source branch name is empty")
    if not to_branch:
        raise ValueError("Target branch name is empty")

    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        # Show changes in from_branch that are not in to_branch
        result = subprocess.run(
            ["git", "diff", "-U5", "--no-color", to_branch, from_branch],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=False,
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


def get_branch_info(path: str = ".") -> dict:
    """Get information about the current branch and available branches.

    Args:
        path: Repository path (default: current directory)

    Returns:
        Dictionary with branch information
    """
    return {
        "current": get_current_branch(path),
        "all": get_all_branches(path),
        "remote": get_remote_branches(path),
    }
