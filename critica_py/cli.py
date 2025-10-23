"""CLI interface for Critica."""

import sys
from pathlib import Path

import click
from rich.console import Console
from rich.panel import Panel
from rich.markdown import Markdown

from critica_py.ai import AIService
from critica_py.config import load_config
from critica_py.git import (
    get_diff,
    is_git_repository,
    get_current_branch,
    get_all_branches,
    get_remote_branches,
    get_branch_diff,
    has_staged_changes,
    stage_all_files,
    create_commit,
    push_branch,
)
from critica_py.git.diff import DiffMode, get_diff_for_mode

console = Console()


def create_ai_service():
    """Create and configure AI service."""
    config = load_config()

    if not config.openai_api_key:
        console.print(
            "[red]Error: OPENAI_API_KEY environment variable not set or missing in config[/red]"
        )
        sys.exit(1)

    return AIService(
        api_key=config.openai_api_key, model=config.openai_model, base_url=config.openai_base_url
    )


def display_analysis_result(result):
    """Display analysis results in a formatted way."""
    console.print("\n[bold cyan]📊 Analysis Results[/bold cyan]")
    console.print("─" * 52)

    if result.summary:
        console.print("\n[bold]📝 Summary:[/bold]")
        console.print(result.summary)

    if result.code_quality:
        console.print("\n[bold]🏆 Code Quality:[/bold]")
        console.print(result.code_quality)

    if result.issues:
        console.print("\n[bold yellow]⚠️  Issues Found:[/bold yellow]")
        for i, issue in enumerate(result.issues, 1):
            console.print(f"  {i}. {issue}")

    if result.improvements:
        console.print("\n[bold green]💡 Improvement Suggestions:[/bold green]")
        for i, improvement in enumerate(result.improvements, 1):
            console.print(f"  {i}. {improvement}")

    if result.security_notes:
        console.print("\n[bold red]🔒 Security Notes:[/bold red]")
        for i, note in enumerate(result.security_notes, 1):
            console.print(f"  {i}. {note}")

    if result.performance_notes:
        console.print("\n[bold magenta]⚡ Performance Notes:[/bold magenta]")
        for i, note in enumerate(result.performance_notes, 1):
            console.print(f"  {i}. {note}")

    if result.commit_message:
        console.print("\n[bold]📝 Suggested Commit Message:[/bold]")
        console.print("─" * 32)
        console.print(result.commit_message)
        console.print("─" * 32)

    if result.pr_description:
        console.print("\n[bold]📋 PR Description:[/bold]")
        console.print("─" * 32)
        console.print(result.pr_description)
        console.print("─" * 32)


@click.group()
@click.version_option(version="2.0.0")
def main():
    """Critica - AI-powered Git analysis CLI tool."""
    pass


@main.command()
@click.argument("path", type=click.Path(exists=True), default=".")
@click.option("--staged", "-s", is_flag=True, help="Analyze only staged changes")
def analyze(path, staged):
    """Perform comprehensive AI analysis of git diff.

    Analyzes the git diff with AI to get insights about:
    - Code quality assessment
    - Potential issues and bugs
    - Security concerns
    - Performance implications
    - Improvement suggestions
    - Explanations of changes
    """
    # Check if we're in a git repository
    if not is_git_repository(path):
        console.print(f"[red]Error: not a git repository: {path}[/red]")
        sys.exit(1)

    # Get the diff
    diff_output = get_diff(path, staged)

    if not diff_output or not diff_output.strip():
        console.print("[yellow]No changes to analyze[/yellow]")
        return

    # Create AI service
    ai_service = create_ai_service()

    console.print("🤖 Analyzing changes with AI...\n")

    # Perform analysis
    try:
        result = ai_service.analyze_diff(diff_output)
        display_analysis_result(result)
    except Exception as e:
        console.print(f"[red]AI analysis failed: {e}[/red]")
        sys.exit(1)


@main.command()
@click.argument("path", type=click.Path(exists=True), default=".")
def commit(path):
    """Generate conventional commit message.

    Generates a conventional commit message based on the git diff.
    Uses AI to analyze changes and create appropriate commit messages
    following conventional commit format.
    """
    # Check if we're in a git repository
    if not is_git_repository(path):
        console.print(f"[red]Error: not a git repository: {path}[/red]")
        sys.exit(1)

    # Check if there are staged changes
    has_staged = has_staged_changes(path)

    if has_staged:
        # Get staged diff
        diff_output = get_diff(path, staged=True)
    else:
        # Check if there are unstaged changes
        unstaged_diff = get_diff_for_mode(path, DiffMode.UNSTAGED)

        if not unstaged_diff or not unstaged_diff.strip():
            console.print("[yellow]No changes to commit[/yellow]")
            return

        # Stage all files automatically
        console.print("No staged changes found. Staging all modified files...")
        try:
            stage_all_files(path)
            console.print("[green]✅ Files staged successfully![/green]")
        except Exception as e:
            console.print(f"[red]Failed to stage files: {e}[/red]")
            sys.exit(1)

        # Get the staged diff after staging
        diff_output = get_diff(path, staged=True)

    if not diff_output or not diff_output.strip():
        console.print("[yellow]No changes to commit[/yellow]")
        return

    # Create AI service
    ai_service = create_ai_service()

    console.print("🤖 Generating commit message...\n")

    # Generate commit message
    try:
        commit_msg = ai_service.generate_commit_message(diff_output)
        console.print("\n")
    except Exception as e:
        console.print(f"[red]Commit message generation failed: {e}[/red]")
        sys.exit(1)

    # Ask for confirmation to apply commit
    if click.confirm("\nDo you want to apply this commit?", default=False):
        # Apply the commit
        console.print("Applying commit...")
        try:
            create_commit(path, commit_msg)
            console.print("[green]✅ Commit applied successfully![/green]\n")
        except Exception as e:
            console.print(f"[red]Failed to create commit: {e}[/red]")
            sys.exit(1)

        # Ask for confirmation to push
        if click.confirm("Do you want to push the branch?", default=False):
            # Push the branch
            console.print("Pushing branch...")
            try:
                push_branch(path)
                console.print("[green]✅ Branch pushed successfully![/green]")
            except Exception as e:
                console.print(f"[red]Failed to push branch: {e}[/red]")
                sys.exit(1)
        else:
            console.print("Push cancelled.")
    else:
        console.print("Commit cancelled.")


@main.command()
@click.argument("path", type=click.Path(exists=True), default=".")
@click.argument("target_branch", required=False)
def pr(path, target_branch):
    """Generate PR description.

    Generates a comprehensive PR description based on the git diff.
    Creates detailed PR descriptions including summary, changes, and testing considerations.

    If no target branch is specified, you will be prompted to select one.
    """
    # Check if we're in a git repository
    if not is_git_repository(path):
        console.print(f"[red]Error: not a git repository: {path}[/red]")
        sys.exit(1)

    # Get current branch
    try:
        current_branch = get_current_branch(path)
    except Exception as e:
        console.print(f"[red]Failed to get current branch: {e}[/red]")
        sys.exit(1)

    # Get all branches
    try:
        local_branches = get_all_branches(path)
        try:
            remote_branches = get_remote_branches(path)
            branches = list(set(local_branches + remote_branches))
        except Exception:
            branches = local_branches
    except Exception as e:
        console.print(f"[red]Failed to get branches: {e}[/red]")
        sys.exit(1)

    if len(branches) < 2:
        console.print("[yellow]Not enough branches for comparison[/yellow]")
        return

    # Show branch selection
    if not target_branch:
        console.print("\n[bold]Available branches:[/bold]")
        for i, branch in enumerate(branches, 1):
            marker = "→ " if branch == current_branch else "  "
            current_marker = " (current)" if branch == current_branch else ""
            console.print(f"{marker}{i}. {branch}{current_marker}")

        console.print(f"\n[cyan]Current branch: {current_branch}[/cyan]")
        console.print(
            "Selecting target branch to compare against (where you want to merge into)\n"
        )

        # Prompt user to select target branch
        selection = click.prompt(
            f"Enter target branch number (1-{len(branches)})", type=int, default=1
        )

        if selection < 1 or selection > len(branches):
            console.print(f"[red]Selection out of range: {selection}[/red]")
            sys.exit(1)

        target_branch = branches[selection - 1]

        # Don't allow comparing branch to itself
        if target_branch == current_branch:
            console.print("[red]Cannot compare branch to itself[/red]")
            sys.exit(1)
    else:
        # Validate that the target branch exists
        if target_branch not in branches:
            console.print(f"[red]Target branch '{target_branch}' not found in available branches[/red]")
            sys.exit(1)

        # Don't allow comparing branch to itself
        if target_branch == current_branch:
            console.print("[red]Cannot compare branch to itself[/red]")
            sys.exit(1)

        console.print(f"Using target branch: {target_branch}")

    console.print(f"\n[cyan]Comparing {current_branch} → {target_branch}[/cyan]\n")

    # Get diff between branches
    try:
        diff_output = get_branch_diff(path, current_branch, target_branch)
    except Exception as e:
        console.print(f"[red]Failed to get branch diff: {e}[/red]")
        sys.exit(1)

    if not diff_output or not diff_output.strip():
        console.print("[yellow]No changes between branches[/yellow]")
        return

    # Create AI service
    ai_service = create_ai_service()

    console.print("🤖 Generating PR description...\n")

    # Generate PR description
    try:
        ai_service.generate_pr_description_with_branches(diff_output, current_branch, target_branch)
        console.print("\n")
        console.print("─" * 52)
    except Exception as e:
        console.print(f"[red]PR description generation failed: {e}[/red]")
        sys.exit(1)


if __name__ == "__main__":
    main()
