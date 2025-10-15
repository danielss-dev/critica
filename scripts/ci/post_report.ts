#!/usr/bin/env -S deno run --allow-net --allow-read --allow-env

/**
 * Post Critica AI review reports to GitHub PR comments or other platforms
 * 
 * Usage:
 *   deno run --allow-net --allow-read --allow-env post_report.ts <report-file> [platform]
 * 
 * Platforms: github (default), gitlab, console
 * 
 * Environment variables:
 *   GITHUB_TOKEN - GitHub personal access token
 *   GITHUB_REPOSITORY - Repository in format owner/repo
 *   GITHUB_PR_NUMBER - Pull request number
 *   GITLAB_TOKEN - GitLab personal access token
 *   GITLAB_PROJECT_ID - GitLab project ID
 *   GITLAB_MR_IID - Merge request IID
 */

interface ReviewReport {
  timestamp: string;
  findings: Finding[];
  summary: string;
  file_count: number;
}

interface Finding {
  severity: string;
  title: string;
  description: string;
  file_path?: string;
  line_number?: number;
  category: string;
}

async function readReport(path: string): Promise<ReviewReport> {
  const content = await Deno.readTextFile(path);
  return JSON.parse(content);
}

function formatMarkdown(report: ReviewReport): string {
  let md = `## 🤖 AI Code Review\n\n`;
  md += `**Summary:** ${report.summary || 'No summary available'}\n\n`;
  md += `**Files analyzed:** ${report.file_count}\n`;
  md += `**Findings:** ${report.findings.length}\n\n`;

  if (report.findings.length > 0) {
    md += `### Findings\n\n`;
    
    for (const finding of report.findings) {
      const icon = finding.severity === 'error' ? '🔴' : 
                   finding.severity === 'warning' ? '🟡' : 'ℹ️';
      
      md += `#### ${icon} ${finding.title}\n\n`;
      md += `**Severity:** \`${finding.severity}\`\n`;
      
      if (finding.file_path) {
        md += `**File:** \`${finding.file_path}\``;
        if (finding.line_number) {
          md += `:${finding.line_number}`;
        }
        md += `\n`;
      }
      
      md += `\n${finding.description}\n\n`;
      md += `---\n\n`;
    }
  }

  md += `\n_Powered by [Critica](https://github.com/danielss-dev/critica)_`;
  return md;
}

async function postToGitHub(report: ReviewReport): Promise<void> {
  const token = Deno.env.get('GITHUB_TOKEN');
  const repo = Deno.env.get('GITHUB_REPOSITORY');
  const prNumber = Deno.env.get('GITHUB_PR_NUMBER');

  if (!token || !repo || !prNumber) {
    throw new Error('Missing GitHub environment variables (GITHUB_TOKEN, GITHUB_REPOSITORY, GITHUB_PR_NUMBER)');
  }

  const [owner, repoName] = repo.split('/');
  const url = `https://api.github.com/repos/${owner}/${repoName}/issues/${prNumber}/comments`;

  const body = formatMarkdown(report);

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Authorization': `token ${token}`,
      'Accept': 'application/vnd.github.v3+json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ body }),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`GitHub API error: ${response.status} - ${error}`);
  }

  console.log('✅ Posted review to GitHub PR');
}

async function postToGitLab(report: ReviewReport): Promise<void> {
  const token = Deno.env.get('GITLAB_TOKEN');
  const projectId = Deno.env.get('GITLAB_PROJECT_ID');
  const mrIid = Deno.env.get('GITLAB_MR_IID');

  if (!token || !projectId || !mrIid) {
    throw new Error('Missing GitLab environment variables (GITLAB_TOKEN, GITLAB_PROJECT_ID, GITLAB_MR_IID)');
  }

  const url = `https://gitlab.com/api/v4/projects/${projectId}/merge_requests/${mrIid}/notes`;

  const body = formatMarkdown(report);

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'PRIVATE-TOKEN': token,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ body }),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`GitLab API error: ${response.status} - ${error}`);
  }

  console.log('✅ Posted review to GitLab MR');
}

function postToConsole(report: ReviewReport): void {
  console.log(formatMarkdown(report));
}

async function main() {
  const args = Deno.args;
  
  if (args.length < 1) {
    console.error('Usage: post_report.ts <report-file> [platform]');
    console.error('Platforms: github (default), gitlab, console');
    Deno.exit(1);
  }

  const reportPath = args[0];
  const platform = args[1] || 'github';

  const report = await readReport(reportPath);

  switch (platform.toLowerCase()) {
    case 'github':
      await postToGitHub(report);
      break;
    case 'gitlab':
      await postToGitLab(report);
      break;
    case 'console':
      postToConsole(report);
      break;
    default:
      throw new Error(`Unknown platform: ${platform}`);
  }
}

if (import.meta.main) {
  try {
    await main();
  } catch (error) {
    console.error(`❌ Error: ${error.message}`);
    Deno.exit(1);
  }
}

