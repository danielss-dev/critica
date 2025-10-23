"""AI service for code analysis using OpenAI."""

import json
import sys
from dataclasses import dataclass
from typing import List, Optional

from openai import OpenAI


@dataclass
class AnalysisResult:
    """AI analysis results."""

    summary: str = ""
    improvements: List[str] = None
    issues: List[str] = None
    explanations: List[str] = None
    commit_message: str = ""
    pr_description: str = ""
    code_quality: str = ""
    security_notes: List[str] = None
    performance_notes: List[str] = None

    def __post_init__(self):
        """Initialize list fields to empty lists if None."""
        if self.improvements is None:
            self.improvements = []
        if self.issues is None:
            self.issues = []
        if self.explanations is None:
            self.explanations = []
        if self.security_notes is None:
            self.security_notes = []
        if self.performance_notes is None:
            self.performance_notes = []


class AIService:
    """AI service for code analysis using OpenAI."""

    def __init__(self, api_key: str, model: str = "gpt-5-nano", base_url: Optional[str] = None):
        """Initialize AI service.

        Args:
            api_key: OpenAI API key
            model: Model to use (default: gpt-5-nano)
            base_url: Optional custom API base URL
        """
        self.model = model
        self.max_completion_tokens = 4000

        if base_url:
            self.client = OpenAI(api_key=api_key, base_url=base_url)
        else:
            self.client = OpenAI(api_key=api_key)

    def analyze_diff(self, diff_content: str) -> AnalysisResult:
        """Perform comprehensive analysis of git diff changes.

        Args:
            diff_content: Raw git diff content

        Returns:
            AnalysisResult with analysis data
        """
        if not diff_content or not diff_content.strip():
            return AnalysisResult()

        prompt = self._build_analysis_prompt(diff_content)

        # Call AI with quiet streaming (don't display raw JSON)
        response = self._call_ai_stream_quiet(prompt)

        # Parse the response
        result = self._parse_analysis_response(response)
        return result

    def generate_commit_message(self, diff_content: str) -> str:
        """Generate a commit message based on the changes.

        Args:
            diff_content: Raw git diff content

        Returns:
            Generated commit message
        """
        if not diff_content or not diff_content.strip():
            return "No changes to commit"

        prompt = self._build_commit_message_prompt(diff_content)
        response = self._call_ai_stream(prompt)

        return response.strip()

    def generate_pr_description(self, diff_content: str) -> str:
        """Generate a PR description based on the changes.

        Args:
            diff_content: Raw git diff content

        Returns:
            Generated PR description
        """
        if not diff_content or not diff_content.strip():
            return "No changes to describe"

        prompt = self._build_pr_description_prompt(diff_content)
        response = self._call_ai_stream(prompt)

        return response.strip()

    def generate_pr_description_with_branches(
        self, diff_content: str, source_branch: str, target_branch: str
    ) -> str:
        """Generate a PR description based on branch diff.

        Args:
            diff_content: Raw git diff content
            source_branch: Source branch name
            target_branch: Target branch name

        Returns:
            Generated PR description
        """
        if not diff_content or not diff_content.strip():
            return "No changes to describe"

        prompt = self._build_pr_description_prompt_with_branches(
            diff_content, source_branch, target_branch
        )
        response = self._call_ai_stream(prompt)

        return response.strip()

    def suggest_improvements(self, diff_content: str) -> List[str]:
        """Provide code improvement suggestions.

        Args:
            diff_content: Raw git diff content

        Returns:
            List of improvement suggestions
        """
        if not diff_content or not diff_content.strip():
            return []

        prompt = self._build_improvements_prompt(diff_content)
        response = self._call_ai_stream(prompt)

        # Parse the response as a list of improvements
        improvements = []
        for line in response.split("\n"):
            line = line.strip()
            if line and not line.startswith("-"):
                improvements.append(line)

        return improvements

    def explain_changes(self, diff_content: str) -> str:
        """Provide explanations for the code changes.

        Args:
            diff_content: Raw git diff content

        Returns:
            Explanation of changes
        """
        if not diff_content or not diff_content.strip():
            return "No changes to explain"

        prompt = self._build_explanation_prompt(diff_content)
        response = self._call_ai_stream(prompt)

        return response.strip()

    def _build_analysis_prompt(self, diff_content: str) -> str:
        """Create a comprehensive analysis prompt."""
        return f"""Analyze the following git diff and provide a comprehensive analysis in JSON format.

IMPORTANT: Return ONLY a single JSON object with these exact fields:
- summary: plain text string (2-3 sentences) - NO NESTED JSON
- improvements: array of strings (improvement suggestions)
- issues: array of strings (potential problems)
- explanations: array of strings (explanations of changes)
- commit_message: plain text string (conventional commit format)
- pr_description: plain text string (multi-line description)
- code_quality: plain text string (quality assessment)
- security_notes: array of strings (security observations)
- performance_notes: array of strings (performance observations)

Focus your analysis on:
1. Code quality and best practices
2. Potential bugs or issues
3. Security concerns
4. Performance implications
5. Code improvements and suggestions
6. Explanations of what changed and why

Git diff:
{diff_content}

RESPOND ONLY WITH VALID JSON - no markdown, no code blocks, no extra text. Each string field must be plain text, never JSON."""

    def _build_commit_message_prompt(self, diff_content: str) -> str:
        """Create a prompt for commit message generation."""
        return f"""Generate a conventional commit message for the following git diff. Use the format:
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]

Types: feat, fix, docs, style, refactor, perf, test, chore, build, ci, revert

Git diff:
{diff_content}

Respond only with the commit message, no additional text."""

    def _build_pr_description_prompt(self, diff_content: str) -> str:
        """Create a prompt for PR description generation."""
        return f"""Generate a comprehensive PR description for the following git diff. Include:

1. Summary of changes
2. What was changed and why
3. Testing considerations
4. Breaking changes (if any)
5. Screenshots or examples (if applicable)

Git diff:
{diff_content}

Respond with a well-formatted PR description."""

    def _build_pr_description_prompt_with_branches(
        self, diff_content: str, source_branch: str, target_branch: str
    ) -> str:
        """Create a prompt for PR description generation with branch context."""
        return f"""Generate a comprehensive PR description for a pull request from branch "{source_branch}" to "{target_branch}". Include:

1. Summary of changes
2. What was changed and why
3. Testing considerations
4. Breaking changes (if any)
5. Screenshots or examples (if applicable)
6. Branch context and merge considerations

Git diff from {source_branch} to {target_branch}:
{diff_content}

Respond with a well-formatted PR description that includes the branch context."""

    def _build_improvements_prompt(self, diff_content: str) -> str:
        """Create a prompt for improvement suggestions."""
        return f"""Analyze the following git diff and provide specific, actionable improvement suggestions. Focus on:

1. Code quality and readability
2. Performance optimizations
3. Security improvements
4. Best practices adherence
5. Error handling
6. Documentation needs

Git diff:
{diff_content}

Provide each suggestion as a separate line, starting with a brief description."""

    def _build_explanation_prompt(self, diff_content: str) -> str:
        """Create a prompt for change explanations."""
        return f"""Explain the following git diff changes in detail. Provide:

1. What each change does
2. Why the change was made (if apparent)
3. Impact of the changes
4. Any potential side effects
5. How the changes relate to each other

Git diff:
{diff_content}

Provide a clear, comprehensive explanation."""

    def _call_ai_stream(self, prompt: str) -> str:
        """Make a streaming request to the AI service and write to stdout.

        Only displays output if stdout is a TTY (interactive terminal).
        """
        should_output = sys.stdout.isatty()
        return self._call_ai_stream_internal(prompt, should_output)

    def _call_ai_stream_quiet(self, prompt: str) -> str:
        """Make a streaming request without writing to stdout."""
        return self._call_ai_stream_internal(prompt, False)

    def _call_ai_stream_internal(self, prompt: str, write_output: bool) -> str:
        """Make a streaming request to the AI service.

        Args:
            prompt: The prompt to send
            write_output: Whether to write output to stdout

        Returns:
            Full response text
        """
        stream = self.client.chat.completions.create(
            model=self.model,
            messages=[{"role": "user", "content": prompt}],
            max_completion_tokens=self.max_completion_tokens,
            stream=True,
        )

        full_response = []

        for chunk in stream:
            if chunk.choices and chunk.choices[0].delta.content:
                content = chunk.choices[0].delta.content
                full_response.append(content)

                if write_output:
                    sys.stdout.write(content)
                    sys.stdout.flush()

        if write_output:
            sys.stdout.write("\n")
            sys.stdout.flush()

        return "".join(full_response)

    def _parse_analysis_response(self, response: str) -> AnalysisResult:
        """Parse the AI response into AnalysisResult."""
        cleaned_response = response.strip()

        # Try to find JSON object in the response
        start_idx = cleaned_response.find("{")
        end_idx = cleaned_response.rfind("}")

        if start_idx == -1 or end_idx == -1 or start_idx >= end_idx:
            # No JSON found, return basic result
            return AnalysisResult(
                summary=response,
                explanations=[response],
                commit_message="Update code",
                pr_description=response,
                code_quality="Unknown",
            )

        # Extract just the JSON part
        json_str = cleaned_response[start_idx : end_idx + 1]

        try:
            data = json.loads(json_str)
        except json.JSONDecodeError:
            # If JSON parsing fails, return basic result
            return AnalysisResult(
                summary=response,
                explanations=[response],
                commit_message="Update code",
                pr_description=response,
                code_quality="Unknown",
            )

        # Extract fields from the dictionary
        result = AnalysisResult(
            summary=self._extract_string(data, "summary"),
            code_quality=self._extract_string(data, "code_quality"),
            commit_message=self._extract_string(data, "commit_message"),
            pr_description=self._extract_string(data, "pr_description"),
            improvements=self._extract_string_array(data, "improvements"),
            issues=self._extract_string_array(data, "issues"),
            explanations=self._extract_string_array(data, "explanations"),
            security_notes=self._extract_string_array(data, "security_notes"),
            performance_notes=self._extract_string_array(data, "performance_notes"),
        )

        # Sanitize all string fields
        result.summary = self._sanitize_field(result.summary, "Summary")
        result.code_quality = self._sanitize_field(result.code_quality, "CodeQuality")
        result.commit_message = self._sanitize_field(result.commit_message, "CommitMessage")
        result.pr_description = self._sanitize_field(result.pr_description, "PR Description")

        # Clean up array items
        result.improvements = self._clean_string_array(result.improvements)
        result.issues = self._clean_string_array(result.issues)
        result.explanations = self._clean_string_array(result.explanations)
        result.security_notes = self._clean_string_array(result.security_notes)
        result.performance_notes = self._clean_string_array(result.performance_notes)

        return result

    def _extract_string(self, data: dict, key: str) -> str:
        """Safely extract a string value from a dictionary."""
        return data.get(key, "")

    def _extract_string_array(self, data: dict, key: str) -> List[str]:
        """Safely extract a string array from a dictionary."""
        value = data.get(key, [])
        if isinstance(value, list):
            return [str(item) for item in value if isinstance(item, str)]
        return []

    def _sanitize_field(self, field: str, field_name: str) -> str:
        """Handle cases where a field contains JSON instead of plain text."""
        if not field:
            return ""

        trimmed = field.strip()

        # Check if field contains JSON (starts with { or [)
        if trimmed.startswith("{") or trimmed.startswith("["):
            try:
                data = json.loads(trimmed)
                if isinstance(data, dict):
                    # Try to extract by common field names
                    for key in ["summary", "description", "message", "content", "text", "value"]:
                        if key in data and isinstance(data[key], str) and data[key]:
                            return data[key]

                    # If it's a simple object with just one string field, return that
                    for value in data.values():
                        if isinstance(value, str) and value:
                            return value

                return f"{field_name} generated successfully"
            except json.JSONDecodeError:
                pass

        return trimmed

    def _clean_string_array(self, arr: List[str]) -> List[str]:
        """Remove empty strings and sanitize array items."""
        result = []
        for item in arr:
            cleaned = self._sanitize_field(item, "Item")
            if cleaned and cleaned != "Item generated successfully":
                result.append(cleaned)
        return result
