"""Git operations for Critica."""

from .diff import get_diff, is_git_repository, get_diff_for_mode, DiffMode
from .branch import (
    get_current_branch, 
    get_branch_info, 
    get_all_branches, 
    get_remote_branches, 
    get_branch_diff
)
from .commit import (
    get_recent_commits, 
    has_staged_changes, 
    stage_all_files, 
    create_commit, 
    push_branch
)

__all__ = [
    "get_diff", 
    "is_git_repository", 
    "get_diff_for_mode", 
    "DiffMode",
    "get_current_branch", 
    "get_branch_info", 
    "get_all_branches", 
    "get_remote_branches", 
    "get_branch_diff",
    "get_recent_commits", 
    "has_staged_changes", 
    "stage_all_files", 
    "create_commit", 
    "push_branch"
]
