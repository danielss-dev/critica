"""Git commit operations."""

import subprocess
from pathlib import Path
from typing import List


def stage_all_files(path: str = ".") -> None:
    """Stage all files in the repository.

    Args:
        path: Repository path (default: current directory)
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        subprocess.run(
            ["git", "add", "."], cwd=work_dir, capture_output=True, text=True, check=True
        )
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to stage all files: {e}")


def create_commit(path: str, message: str) -> None:
    """Create a commit with the given message.

    Args:
        path: Repository path
        message: Commit message
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        subprocess.run(
            ["git", "commit", "-m", message],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to create commit: {e}")


def has_staged_changes(path: str = ".") -> bool:
    """Check if there are any staged changes.

    Args:
        path: Repository path (default: current directory)

    Returns:
        True if there are staged changes, False otherwise
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        result = subprocess.run(
            ["git", "diff", "--staged", "--quiet"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=False,
        )
        # Exit code 1 means there are staged changes
        # Exit code 0 means no staged changes
        return result.returncode == 1
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to check staged changes: {e}")


def get_staged_files(path: str = ".") -> List[str]:
    """Get list of staged files.

    Args:
        path: Repository path (default: current directory)

    Returns:
        List of staged file paths
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        result = subprocess.run(
            ["git", "diff", "--staged", "--name-only"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )

        files = [f for f in result.stdout.strip().split("\n") if f]
        return files
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to get staged files: {e}")


def push_branch(path: str = ".") -> None:
    """Push the current branch to the remote repository.

    Args:
        path: Repository path (default: current directory)
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        subprocess.run(["git", "push"], cwd=work_dir, capture_output=True, text=True, check=True)
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to push branch: {e}")


def get_recent_commits(path: str = ".", count: int = 5) -> List[dict]:
    """Get recent commit messages.

    Args:
        path: Repository path (default: current directory)
        count: Number of commits to retrieve (default: 5)

    Returns:
        List of dictionaries with commit information
    """
    abs_path = Path(path).resolve()

    if abs_path.is_file():
        work_dir = abs_path.parent
    else:
        work_dir = abs_path

    try:
        # Format: hash|author|date|message
        result = subprocess.run(
            ["git", "log", f"-{count}", "--pretty=format:%h|%an|%ar|%s"],
            cwd=work_dir,
            capture_output=True,
            text=True,
            check=True,
        )

        commits = []
        for line in result.stdout.strip().split("\n"):
            if not line:
                continue
            parts = line.split("|", 3)
            if len(parts) == 4:
                commits.append(
                    {"hash": parts[0], "author": parts[1], "date": parts[2], "message": parts[3]}
                )

        return commits
    except subprocess.CalledProcessError:
        return []
