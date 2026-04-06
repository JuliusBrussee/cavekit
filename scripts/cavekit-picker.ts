#!/usr/bin/env npx tsx

import { checkbox, Separator } from "@inquirer/prompts";
import { readdirSync, readFileSync, existsSync, writeFileSync } from "fs";
import { basename, join, resolve } from "path";
import { execFileSync } from "child_process";

// ANSI helpers
const dim = (s: string) => `\x1b[2m${s}\x1b[22m`;
const yellow = (s: string) => `\x1b[33m${s}\x1b[39m`;
const strikethrough = (s: string) => `\x1b[9m${s}\x1b[29m`;

interface FrontierInfo {
  path: string;
  name: string;
  totalTasks: number;
  doneTasks: number;
  status: "available" | "in-progress" | "done";
}

function getProjectRoot(): string {
  try {
    return execFileSync("git", ["rev-parse", "--show-toplevel"], {
      encoding: "utf8",
    }).trim();
  } catch {
    return process.cwd();
  }
}

function discoverFrontiers(projectRoot: string): FrontierInfo[] {
  // Try context/plans first (current convention), fall back to context/sites (legacy)
  let frontiersDir = join(projectRoot, "context/plans");
  if (!existsSync(frontiersDir)) {
    frontiersDir = join(projectRoot, "context/sites");
    if (!existsSync(frontiersDir)) {
      return [];
    }
  }

  const results: FrontierInfo[] = [];

  // Active frontiers
  const activeFiles = readdirSync(frontiersDir).filter(
    (f) => f.endsWith(".md")
  );

  for (const file of activeFiles) {
    const fullPath = join(frontiersDir, file);
    const content = readFileSync(fullPath, "utf8");
    const name = deriveName(file);

    const taskLines = content.match(/\|\s*T-(?:[A-Za-z0-9]+-)*\d+\s*\|/g) || [];
    const totalTasks = taskLines.length;
    const doneTasks = countDoneTasks(projectRoot, content);
    const status = detectStatus(projectRoot, name, totalTasks, doneTasks);

    results.push({ path: fullPath, name, totalTasks, doneTasks, status });
  }

  // Archived frontiers — always shown as done
  const archiveDir = join(frontiersDir, "archive");
  if (existsSync(archiveDir)) {
    const archivedFiles = readdirSync(archiveDir).filter(
      (f) => f.endsWith(".md")
    );
    for (const file of archivedFiles) {
      const fullPath = join(archiveDir, file);
      const content = readFileSync(fullPath, "utf8");
      const name = deriveName(file);

      const taskLines = content.match(/\|\s*T-(?:[A-Za-z0-9]+-)*\d+\s*\|/g) || [];
      const totalTasks = taskLines.length;

      results.push({
        path: fullPath,
        name,
        totalTasks,
        doneTasks: totalTasks,
        status: "done",
      });
    }
  }

  return results;
}

function deriveName(filename: string): string {
  return filename
    .replace(/\.md$/, "")
    .replace(/^(plan-|feature-frontier-|feature-|build-site-)/, "")
    .replace(/-?frontier-?/, "")
    .replace(/^-|-$/g, "");
}

function scanImplFilesForDone(dir: string, doneSet: Set<string>): void {
  try {
    const entries = readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = join(dir, entry.name);
      if (entry.isDirectory()) {
        // Recurse into archive subdirectories
        scanImplFilesForDone(fullPath, doneSet);
      } else if (entry.name.startsWith("impl-") && entry.name.endsWith(".md")) {
        const content = readFileSync(fullPath, "utf8");
        const matches = content.matchAll(/\b(T-(?:[A-Za-z0-9]+-)*\d+)\b.*?\bDONE\b/gi);
        for (const m of matches) {
          doneSet.add(m[1]);
        }
      }
    }
  } catch {
    // Directory doesn't exist or isn't readable
  }
}

function countDoneTasks(projectRoot: string, frontierContent: string): number {
  const doneSet = new Set<string>();

  // Scan current impl dir + archives
  scanImplFilesForDone(join(projectRoot, "context/impl"), doneSet);

  // Also scan any worktree impl dirs for this project
  const projectName = basename(projectRoot);
  try {
    const parentDir = resolve(projectRoot, "..");
    const siblings = readdirSync(parentDir);
    for (const sibling of siblings) {
      if (sibling.startsWith(`${projectName}-cavekit-`)) {
        scanImplFilesForDone(join(parentDir, sibling, "context/impl"), doneSet);
      }
    }
  } catch {
    // Parent dir not readable
  }

  // Count how many of THIS frontier's tasks are done
  const taskIds = frontierContent.match(/T-(?:[A-Za-z0-9]+-)*\d+/g) || [];
  const uniqueIds = new Set(taskIds);
  let count = 0;
  for (const id of uniqueIds) {
    if (doneSet.has(id)) count++;
  }
  return count;
}

function detectStatus(
  projectRoot: string,
  name: string,
  totalTasks: number,
  doneTasks: number
): "available" | "in-progress" | "done" {
  if (totalTasks > 0 && doneTasks >= totalTasks) return "done";

  // Check if a worktree exists for this frontier
  const projectName = basename(projectRoot);
  const worktreePath = resolve(projectRoot, `../${projectName}-cavekit-${name}`);
  if (existsSync(worktreePath)) {
    // Check if ralph loop is active in the worktree
    const ralphState = join(worktreePath, ".claude/ralph-loop.local.md");
    if (existsSync(ralphState)) return "in-progress";
    // Worktree exists but no active loop — could be finished or stale
    if (doneTasks > 0) return "in-progress";
  }

  return "available";
}

function formatChoice(f: FrontierInfo): string {
  const progress = f.totalTasks > 0 ? `${f.doneTasks}/${f.totalTasks}` : "?";

  switch (f.status) {
    case "done":
      return `${strikethrough(f.name)} ${dim(`(${progress} done)`)}`;
    case "in-progress":
      return `${yellow("⟳")} ${f.name} ${dim(`(${progress})`)}`;
    case "available":
      return `${f.name} ${dim(`(${progress} tasks)`)}`;
  }
}

async function main() {
  const projectRoot = getProjectRoot();
  const frontiers = discoverFrontiers(projectRoot);

  if (frontiers.length === 0) {
    console.error("No frontiers found in context/plans/ or context/sites/");
    console.error("Run /ck:map first to generate one.");
    process.exit(1);
  }

  const available = frontiers.filter((f) => f.status === "available");
  const inProgress = frontiers.filter((f) => f.status === "in-progress");
  const done = frontiers.filter((f) => f.status === "done");

  type ChoiceOrSep =
    | {
        name: string;
        value: string;
        disabled?: string | boolean;
        checked?: boolean;
      }
    | Separator;
  const choices: ChoiceOrSep[] = [];

  if (available.length > 0) {
    choices.push(new Separator(dim("── Available ──")));
    for (const f of available) {
      choices.push({
        name: formatChoice(f),
        value: f.path,
        checked: true,
      });
    }
  }

  if (inProgress.length > 0) {
    choices.push(new Separator(dim("── In Progress (select to resume) ──")));
    for (const f of inProgress) {
      choices.push({
        name: formatChoice(f),
        value: f.path,
      });
    }
  }

  if (done.length > 0) {
    choices.push(new Separator(dim("── Done ──")));
    for (const f of done) {
      choices.push({
        name: formatChoice(f),
        value: f.path,
        disabled: "complete",
      });
    }
  }

  try {
    const selected = await checkbox({
      message: "Select build sites to launch",
      choices,
      instructions: false,
      theme: {
        style: {
          keysHelpTip: () =>
            dim("[space] toggle  [a] all  [enter] launch  [ctrl+c] quit"),
        },
      },
    });

    if (selected.length === 0) {
      console.error("No frontiers selected.");
      process.exit(1);
    }

    // Write selected paths to outfile (env var) or stdout
    const outfile = process.env.CAVEKIT_PICKER_OUTFILE;
    const output = selected.join("\n") + "\n";
    if (outfile) {
      writeFileSync(outfile, output);
    } else {
      process.stdout.write(output);
    }
  } catch {
    // User pressed Ctrl+C
    process.exit(1);
  }
}

main();
