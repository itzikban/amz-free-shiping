/**
 * @process branch-merge-align
 * @description Merge and align feat/fill-to-50-basket-booster into ITZ-54-feat/fill-to-50-basket-booster.
 * Analyzes diff, performs git merge, resolves conflicts, validates build, and pushes.
 * @inputs { sourceBranch: string, destBranch: string, repoDir: string }
 * @outputs { success: boolean, conflictsResolved: number, commitSha: string }
 *
 * @agent conflict-resolver specializations/web-development/agents/fullstack-architect/AGENT.md
 */

import { defineTask } from '@a5c-ai/babysitter-sdk';

export async function process(inputs, ctx) {
  const {
    sourceBranch = 'origin/feat/fill-to-50-basket-booster',
    destBranch = 'ITZ-54-feat/fill-to-50-basket-booster',
    repoDir = '/home/ubuntu/.openclaw/workspace/amz-free-shiping',
  } = inputs || {};

  ctx.log('info', `Branch alignment: merging ${sourceBranch} -> ${destBranch}`);

  // ============================================================================
  // PHASE 1: ANALYSIS
  // ============================================================================
  ctx.log('info', 'Phase 1: Analysing branch differences');

  const analysis = await ctx.task(analyseTask, { sourceBranch, destBranch, repoDir });

  // ============================================================================
  // PHASE 2: MERGE + CONFLICT RESOLUTION
  // ============================================================================
  ctx.log('info', 'Phase 2: Merge and conflict resolution');

  const mergeResult = await ctx.task(mergeTask, {
    sourceBranch,
    destBranch,
    repoDir,
    analysis,
  });

  // ============================================================================
  // PHASE 3: VALIDATION
  // ============================================================================
  ctx.log('info', 'Phase 3: Build validation');

  const validation = await ctx.task(validateTask, { repoDir, mergeResult });

  // ============================================================================
  // PHASE 4: PUSH
  // ============================================================================
  ctx.log('info', 'Phase 4: Push aligned branch');

  const pushResult = await ctx.task(pushTask, { repoDir, destBranch, validation });

  ctx.log('info', `Done. Branch aligned and pushed: ${pushResult.commitSha}`);
  return { success: true, ...pushResult };
}

// ---------------------------------------------------------------------------
// Task: Analyse
// ---------------------------------------------------------------------------
const analyseTask = defineTask('analyse-branch-diff', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Analyse branch differences',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Git expert and full-stack developer',
      task: 'Analyse the exact commits and file-level differences that feat/fill-to-50-basket-booster has compared to ITZ-54-feat/fill-to-50-basket-booster, and produce a structured plan for what needs to merge.',
      context: {
        repoDir: args.repoDir,
        sourceBranch: args.sourceBranch,
        destBranch: args.destBranch,
      },
      instructions: [
        'cd to repoDir and run: git fetch origin',
        'Run: git log --oneline $(git merge-base origin/feat/fill-to-50-basket-booster origin/ITZ-54-feat/fill-to-50-basket-booster)..origin/feat/fill-to-50-basket-booster',
        'Run: git diff --stat $(git merge-base origin/feat/fill-to-50-basket-booster origin/ITZ-54-feat/fill-to-50-basket-booster)..origin/feat/fill-to-50-basket-booster',
        'List every file that both branches modified (overlap files that will conflict)',
        'Output a structured JSON summary with: commitCount, files, overlapFiles, and a human-readable mergeStrategy string',
      ],
      outputFormat: 'JSON with keys: commitCount, files, overlapFiles, mergeStrategy',
    },
    outputSchema: {
      type: 'object',
      required: ['commitCount', 'files', 'overlapFiles', 'mergeStrategy'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Merge + conflict resolution
// ---------------------------------------------------------------------------
const mergeTask = defineTask('merge-and-resolve', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Merge feat/ into ITZ-54-feat/ and resolve conflicts',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Senior full-stack developer and Git expert',
      task: 'Checkout ITZ-54-feat/fill-to-50-basket-booster, merge origin/feat/fill-to-50-basket-booster into it, and intelligently resolve any conflicts — preserving all functionality from both branches.',
      context: {
        repoDir: args.repoDir,
        sourceBranch: args.sourceBranch,
        destBranch: args.destBranch,
        analysis: args.analysis,
      },
      instructions: [
        'cd to repoDir',
        'Run: git checkout ITZ-54-feat/fill-to-50-basket-booster && git pull origin ITZ-54-feat/fill-to-50-basket-booster',
        'Run: git merge origin/feat/fill-to-50-basket-booster --no-edit',
        'If merge succeeds with no conflicts: skip to commit step',
        'If there are conflicts: read EACH conflicted file carefully. For EACH conflict:',
        '  - ITZ-54-feat is the primary branch (more up-to-date, has more features)',
        '  - feat/ adds: ranking improvements, clickable suggestions, captcha noise fixes',
        '  - Keep ALL features from BOTH sides — never discard code, merge intelligently',
        '  - NEVER leave conflict markers (<<<<<<, =======, >>>>>>>) in the files',
        'After resolving: git add <resolved-files>',
        'Run: git diff --check to verify NO conflict markers remain',
        'Run: git commit --no-edit (or git merge --continue)',
        'Return the new HEAD sha and list of resolved files',
      ],
      outputFormat: 'JSON with keys: success, headSha, resolvedFiles, conflictCount',
    },
    outputSchema: {
      type: 'object',
      required: ['success', 'headSha'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Validation
// ---------------------------------------------------------------------------
const validateTask = defineTask('validate-build', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Validate frontend build and backend tests',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'QA engineer and build validator',
      task: 'Run frontend build and backend tests in the repo. Fix any compilation errors introduced by the merge.',
      context: {
        repoDir: args.repoDir,
        mergeResult: args.mergeResult,
      },
      instructions: [
        'cd to repoDir/frontend',
        'Run: rm -rf .next && npm run build 2>&1',
        'If build fails: analyze the error, fix it (edit the relevant file), then re-run build',
        'cd to repoDir/backend',
        'Run: go build ./... 2>&1',
        'Run: go test ./... 2>&1 (skip if only frontend files changed)',
        'Report build status, any errors found and fixed',
        'Return JSON: { frontendBuildOk: bool, backendBuildOk: bool, fixesApplied: string[] }',
      ],
      outputFormat: 'JSON with keys: frontendBuildOk, backendBuildOk, fixesApplied',
    },
    outputSchema: {
      type: 'object',
      required: ['frontendBuildOk', 'backendBuildOk'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Push
// ---------------------------------------------------------------------------
const pushTask = defineTask('push-aligned-branch', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Push aligned branch to origin',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Git operator',
      task: 'Push the aligned ITZ-54-feat/fill-to-50-basket-booster branch to origin and report the final commit SHA.',
      context: {
        repoDir: args.repoDir,
        destBranch: args.destBranch,
        validation: args.validation,
      },
      instructions: [
        'cd to repoDir',
        'Run: git push origin ITZ-54-feat/fill-to-50-basket-booster',
        'Run: git log --oneline -3 to confirm push',
        'Return JSON: { pushed: bool, commitSha: string, summary: string }',
      ],
      outputFormat: 'JSON with keys: pushed, commitSha, summary',
    },
    outputSchema: {
      type: 'object',
      required: ['pushed', 'commitSha'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));
