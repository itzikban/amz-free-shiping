/**
 * @process fix-fill-images-links
 * @description Fix fill-to-threshold suggestion cards: broken images and non-clickable links for NL+other countries.
 * Also adds E2E tests, wires them into auto-launch.sh sanity check, and creates GitHub CI workflow.
 * @inputs { repoDir: string }
 * @outputs { success: boolean, fixesSummary: string }
 *
 * @agent backend-fixer specializations/backend-development/agents/backend-architect/AGENT.md
 * @agent frontend-fixer specializations/web-development/agents/fullstack-architect/AGENT.md
 */

import { defineTask } from '@a5c-ai/babysitter-sdk';

export async function process(inputs, ctx) {
  const { repoDir = '/home/ubuntu/.openclaw/workspace/amz-free-shiping' } = inputs || {};

  ctx.log('info', 'Starting fill-to-threshold bug fix process');

  // ============================================================================
  // PHASE 1: DIAGNOSE
  // ============================================================================
  ctx.log('info', 'Phase 1: Diagnose live API response');

  const diagnosis = await ctx.task(diagnoseTask, { repoDir });

  // ============================================================================
  // PHASE 2: FIX BACKEND
  // ============================================================================
  ctx.log('info', 'Phase 2: Fix backend — URL domain + image URL fallback');

  const backendFix = await ctx.task(fixBackendTask, { repoDir, diagnosis });

  // ============================================================================
  // PHASE 3: FIX FRONTEND
  // ============================================================================
  ctx.log('info', 'Phase 3: Fix frontend — fallback URLs and image onError');

  const frontendFix = await ctx.task(fixFrontendTask, { repoDir, backendFix });

  // ============================================================================
  // PHASE 4: BUILD VALIDATION
  // ============================================================================
  ctx.log('info', 'Phase 4: Build validation');

  const buildResult = await ctx.task(buildValidateTask, { repoDir });

  // ============================================================================
  // PHASE 5: E2E TESTS
  // ============================================================================
  ctx.log('info', 'Phase 5: Create integration/E2E tests');

  const testResult = await ctx.task(createTestsTask, { repoDir, diagnosis });

  // ============================================================================
  // PHASE 6: UPDATE AUTO-LAUNCH.SH WITH SANITY CHECK
  // ============================================================================
  ctx.log('info', 'Phase 6: Wire tests into auto-launch.sh sanity check');

  const launchResult = await ctx.task(updateAutoLaunchTask, { repoDir, testResult });

  // ============================================================================
  // PHASE 7: GITHUB CI WORKFLOW
  // ============================================================================
  ctx.log('info', 'Phase 7: Create GitHub Actions CI workflow');

  const ciResult = await ctx.task(createCIWorkflowTask, { repoDir });

  // ============================================================================
  // PHASE 8: FINAL VERIFY WITH AUTO-LAUNCH
  // ============================================================================
  ctx.log('info', 'Phase 8: Run auto-launch.sh + sanity tests');

  const verifyResult = await ctx.task(finalVerifyTask, { repoDir });

  // ============================================================================
  // PHASE 9: COMMIT AND PUSH
  // ============================================================================
  ctx.log('info', 'Phase 9: Commit and push all changes');

  const pushResult = await ctx.task(commitAndPushTask, { repoDir, verifyResult });

  ctx.log('info', `Done. SHA: ${pushResult.commitSha}`);
  return { success: true, fixesSummary: pushResult.summary };
}

// ---------------------------------------------------------------------------
// Task: Diagnose
// ---------------------------------------------------------------------------
const diagnoseTask = defineTask('diagnose-api', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Diagnose live API — check what backend actually returns for NL scenario',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Backend debugger',
      task: 'Start the backend and call the fill-to-threshold API with a real NL Amazon URL to capture the exact JSON response, focusing on url and image_url fields.',
      context: { repoDir: args.repoDir },
      instructions: [
        'cd to repoDir/backend',
        'Check if backend is running: curl -sf http://127.0.0.1:8085/health || echo "not running"',
        'If not running: cd repoDir && go run ./cmd/server & sleep 5',
        'Test with a simple NL-like Amazon URL: curl -s "http://127.0.0.1:8085/api/v1/fill-to-threshold?url=https://www.amazon.nl/dp/B08N5LNQCX&country=NL&threshold=50" | python3 -m json.tool 2>&1 | head -80',
        'Also test with US URL: curl -s "http://127.0.0.1:8085/api/v1/fill-to-threshold?url=https://www.amazon.com/dp/B08N5LNQCX&country=US&threshold=50" | python3 -m json.tool 2>&1 | head -80',
        'In the response JSON, check for: suggestions[].url (is it populated? is it amazon.com or amazon.nl?), suggestions[].image_url (is it populated?)',
        'Also check the source code: grep -n "amazon.com/dp" backend/internal/checker/checker.go to find hardcoded domain',
        'Return diagnosis: { urlFieldPopulated: bool, imageUrlFieldPopulated: bool, hardcodedDomainLines: [], nlUrlCorrect: bool, rootCause: string }',
      ],
      outputFormat: 'JSON with keys: urlFieldPopulated, imageUrlFieldPopulated, hardcodedDomainLines, nlUrlCorrect, rootCause',
    },
    outputSchema: {
      type: 'object',
      required: ['rootCause'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Fix Backend
// ---------------------------------------------------------------------------
const fixBackendTask = defineTask('fix-backend', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Fix backend — use correct domain for alt URLs, add image fallback',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Backend Go developer',
      task: 'Fix two bugs in the backend checker: (1) scraper hardcodes amazon.com for alt product URLs instead of using the source domain; (2) decodo markdown parser returns empty image_url when Amazon page lacks images in the similar-items section.',
      context: { repoDir: args.repoDir, diagnosis: args.diagnosis },
      instructions: [
        'Open backend/internal/checker/checker.go',
        'FIND the line: `URL: fmt.Sprintf("https://amazon.com/dp/%s", asin)` (around line 344)',
        'FIX: This function (fetchAlternativesFromAmazon or similar) needs to accept a domain parameter OR use the product URL\'s domain. Change hardcoded "amazon.com" to use the passed domain. Check the function signature and how it\'s called — if domain is not passed, add it as a parameter.',
        'Open backend/internal/checker/decodo.go',
        'In extractAlternativesFromMarkdown: after imgURL extraction, if imgURL is still empty, construct a fallback Amazon product image URL using the ASIN format: "https://images-na.ssl-images-amazon.com/images/I/{asin}._AC_SL500_.jpg" — note this won\'t always work but is better than nothing',
        'Actually better fallback: use the ASIN to build a direct Amazon product page image: just leave imgURL empty for now if not found, but add a TODO comment. The real fix is to ensure Decodo returns images.',
        'Run: cd repoDir/backend && go build ./... 2>&1 to verify no compile errors',
        'Run: go test ./... 2>&1',
        'Return: { fixed: bool, filesChanged: [], buildPassed: bool }',
      ],
      outputFormat: 'JSON with keys: fixed, filesChanged, buildPassed',
    },
    outputSchema: {
      type: 'object',
      required: ['fixed', 'filesChanged', 'buildPassed'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Fix Frontend
// ---------------------------------------------------------------------------
const fixFrontendTask = defineTask('fix-frontend', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Fix frontend — fallback item URLs and graceful image error handling',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Frontend React/Next.js developer',
      task: 'Fix two UI bugs: (1) fallback fill-to-threshold items (ft1/ft2/ft3) have url:undefined making them non-clickable; (2) images show blank when image_url loads but URL is broken.',
      context: { repoDir: args.repoDir },
      instructions: [
        'Open frontend/app/page.tsx',
        'FIND the fallbackFillToThresholdSuggestions array (around line 200)',
        'FIX: Give each fallback item a real Amazon search URL based on its title. Use format: `url: "https://www.amazon.com/s?k=" + encodeURIComponent(title)` — but since title uses t() (i18n), build the URL using a generic search term. Better approach: use a hardcoded search URL that makes sense, e.g.: ft1 → "https://www.amazon.com/s?k=books+kindle" (affordable items). Or just set a generic "https://www.amazon.com" as a minimum.',
        'ACTUALLY best fix: for fallback items, build URL as `https://www.amazon.com/s?k=${encodeURIComponent(s.title)}` — but since title is from t() and computed at render time, you need to do this at the point where fillToThresholdSuggestions is rendered, not in the static array.',
        'Alternative clean fix: in the recommendation card rendering (around line 306), if `!s.url` but we have a title, set `href` to `https://www.amazon.com/s?k=${encodeURIComponent(s.title)}` so it\'s always clickable.',
        'FIX images: for the <img> tags rendering suggestion images (around line 310-311), add an onError handler: `onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}` — so when image fails to load it hides instead of showing broken icon.',
        'ALSO: for the outer <img> tags around line 310, wrap them in a div with the img-placeholder-2 class as fallback via CSS or use a state approach.',
        'Simplest reliable fix for images: use `onError={(e) => { const t = e.target as HTMLImageElement; t.src = ""; t.style.display="none"; (t.parentElement as HTMLElement).classList.add("img-placeholder-2"); }}`',
        'Run: cd repoDir/frontend && npm run build 2>&1 to verify',
        'Return: { fixed: bool, filesChanged: [], buildPassed: bool }',
      ],
      outputFormat: 'JSON with keys: fixed, filesChanged, buildPassed',
    },
    outputSchema: {
      type: 'object',
      required: ['fixed', 'filesChanged', 'buildPassed'],
    },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Build Validate
// ---------------------------------------------------------------------------
const buildValidateTask = defineTask('build-validate', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Full build validation — frontend + backend',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Build validator',
      task: 'Run complete build validation for both frontend and backend. Fix any errors.',
      context: { repoDir: args.repoDir },
      instructions: [
        'cd repoDir/frontend && rm -rf .next && npm run build 2>&1',
        'If build fails: read the error, fix the relevant file, re-run',
        'cd repoDir/backend && go build ./... 2>&1',
        'cd repoDir/backend && go test ./... 2>&1',
        'Return: { frontendOk: bool, backendOk: bool, fixesApplied: [] }',
      ],
      outputFormat: 'JSON with keys: frontendOk, backendOk, fixesApplied',
    },
    outputSchema: { type: 'object', required: ['frontendOk', 'backendOk'] },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Create E2E Tests
// ---------------------------------------------------------------------------
const createTestsTask = defineTask('create-tests', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Create E2E and integration tests for fill-to-threshold suggestion cards',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'QA engineer writing integration and E2E tests',
      task: 'Create integration tests and/or E2E tests that verify: (1) the fill-to-threshold API returns suggestions with valid urls and image_urls; (2) the UI renders clickable card links (not plain articles); (3) images render without 404/error.',
      context: { repoDir: args.repoDir, diagnosis: args.diagnosis },
      instructions: [
        'First check: does the project have a test framework? Run: ls repoDir/frontend/package.json repoDir/backend/*_test.go 2>/dev/null',
        'Check package.json for test scripts: grep -i "test\\|jest\\|playwright\\|cypress" repoDir/frontend/package.json',
        'STRATEGY: Create simple shell-based integration tests in scripts/sanity-test.sh that:',
        '  1. Check backend health: curl -sf http://127.0.0.1:8085/health',
        '  2. Check frontend health: curl -sf http://127.0.0.1:3000/api/health | grep -q "ok"',
        '  3. Call fill-to-threshold API with a test URL and verify response has suggestions with non-empty url fields',
        '  4. Verify frontend page loads: curl -sf http://127.0.0.1:3000 | grep -q "Amazon Free Shipping"',
        'Also create backend/internal/checker/fill_to_threshold_integration_test.go with a mock test that verifies suggestion URLs are valid Amazon URLs (not empty)',
        'Make scripts/sanity-test.sh executable (chmod +x)',
        'Run the sanity tests: bash scripts/sanity-test.sh 2>&1',
        'Return: { testsCreated: [], sanityScriptPath: string, allTestsPassed: bool }',
      ],
      outputFormat: 'JSON with keys: testsCreated, sanityScriptPath, allTestsPassed',
    },
    outputSchema: { type: 'object', required: ['testsCreated', 'sanityScriptPath'] },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Update auto-launch.sh
// ---------------------------------------------------------------------------
const updateAutoLaunchTask = defineTask('update-auto-launch', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Wire sanity tests into auto-launch.sh as a post-launch health check',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'DevOps engineer',
      task: 'Update scripts/auto-launch.sh to run the sanity tests after services come up, so the user knows everything works before pushing.',
      context: { repoDir: args.repoDir, testResult: args.testResult },
      instructions: [
        'Read scripts/auto-launch.sh',
        'At the end of the main() function (before the final log "Done"), add:',
        '  if [[ -f "$ROOT_DIR/scripts/sanity-test.sh" ]]; then',
        '    log "Running sanity tests..."',
        '    if bash "$ROOT_DIR/scripts/sanity-test.sh"; then',
        '      log "Sanity tests PASSED ✓"',
        '    else',
        '      log "WARNING: Sanity tests FAILED — check logs above"',
        '    fi',
        '  fi',
        'Do NOT make the sanity test failure cause auto-launch.sh to exit — use warning only (tests fail when services are warming up)',
        'Save the file',
        'Return: { updated: bool, lineAdded: string }',
      ],
      outputFormat: 'JSON with keys: updated, lineAdded',
    },
    outputSchema: { type: 'object', required: ['updated'] },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Create GitHub CI Workflow
// ---------------------------------------------------------------------------
const createCIWorkflowTask = defineTask('create-ci-workflow', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Create GitHub Actions CI workflow for PR validation',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'DevOps/CI engineer',
      task: 'Create a GitHub Actions workflow that runs on PRs to main and validates: frontend build, backend build+tests, and linting. Must not break critical actions.',
      context: { repoDir: args.repoDir },
      instructions: [
        'Create directory: mkdir -p repoDir/.github/workflows',
        'Create file: repoDir/.github/workflows/pr-validation.yml',
        'The workflow should:',
        '  - Trigger on: pull_request targeting main (and push to main)',
        '  - Jobs:',
        '    1. frontend-build: checkout, node 20, npm ci, npm run build (in frontend/)',
        '    2. backend-build: checkout, go 1.22, go build ./..., go test ./... (in backend/)',
        '    3. lint: (optional) run go vet ./... and tsc --noEmit',
        '  - Each job should fail the PR if it fails',
        '  - Use caching for node_modules and Go modules',
        '  - Keep it simple and reliable — no complex matrix builds',
        'Save the file',
        'Return: { created: bool, workflowPath: string }',
      ],
      outputFormat: 'JSON with keys: created, workflowPath',
    },
    outputSchema: { type: 'object', required: ['created', 'workflowPath'] },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Final Verify
// ---------------------------------------------------------------------------
const finalVerifyTask = defineTask('final-verify', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Run auto-launch.sh to start services and verify sanity tests pass',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'QA verification engineer',
      task: 'Run auto-launch.sh to start/restart services and then run the sanity tests to confirm everything works before pushing.',
      context: { repoDir: args.repoDir },
      instructions: [
        'cd repoDir',
        'Run: bash scripts/auto-launch.sh 2>&1 | tail -30',
        'Wait for services to be up (auto-launch.sh waits internally)',
        'Run: bash scripts/sanity-test.sh 2>&1',
        'Check that sanity tests pass',
        'If any test fails: investigate and fix the root cause',
        'Return: { servicesUp: bool, sanityPassed: bool, details: string }',
      ],
      outputFormat: 'JSON with keys: servicesUp, sanityPassed, details',
    },
    outputSchema: { type: 'object', required: ['servicesUp', 'sanityPassed'] },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));

// ---------------------------------------------------------------------------
// Task: Commit and Push
// ---------------------------------------------------------------------------
const commitAndPushTask = defineTask('commit-push', (args, taskCtx) => ({
  kind: 'agent',
  title: 'Commit all fixes and push to ITZ-54-feat/fill-to-50-basket-booster',
  agent: {
    name: 'fullstack-architect',
    prompt: {
      role: 'Git operator',
      task: 'Stage all changed files, create a descriptive commit, and push to origin.',
      context: { repoDir: args.repoDir, verifyResult: args.verifyResult },
      instructions: [
        'cd repoDir',
        'Run: git diff --check to verify no conflict markers',
        'Run: git status to see all changed files',
        'Stage the relevant files: git add backend/internal/checker/checker.go backend/internal/checker/decodo.go frontend/app/page.tsx scripts/sanity-test.sh scripts/auto-launch.sh .github/workflows/pr-validation.yml (and any other changed files)',
        'Do NOT stage .next/, node_modules/, or binary files',
        'Commit: git commit -m "fix(fill-ui): fix broken images and non-clickable suggestion links\\n\\n- Fix checker.go hardcoded amazon.com domain for alt URLs\\n- Add image onError fallback in frontend cards\\n- Add Amazon search URL for fallback suggestion items\\n- Add sanity-test.sh for quick pre-push validation\\n- Wire sanity tests into auto-launch.sh\\n- Add GitHub Actions PR validation workflow"',
        'Push: git push origin ITZ-54-feat/fill-to-50-basket-booster',
        'Return: { pushed: bool, commitSha: string, summary: string }',
      ],
      outputFormat: 'JSON with keys: pushed, commitSha, summary',
    },
    outputSchema: { type: 'object', required: ['pushed', 'commitSha'] },
  },
  io: {
    inputJsonPath: `tasks/${taskCtx.effectId}/input.json`,
    outputJsonPath: `tasks/${taskCtx.effectId}/result.json`,
  },
}));
