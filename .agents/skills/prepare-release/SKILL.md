---
name: prepare-release
description: Prepare or publish a versioned release of recipe-extractor, including version selection, release checks, notes, an annotated Git tag, a GitHub release, and Docker publishing verification. Use for project release requests, not npm package publishing.
---

# Prepare release

Read root `AGENTS.md` and the current `.github/workflows/docker-publish.yml`. Complete preparation before any publishing step. Keep the user's requested version, target, and authorization across the workflow.

## Choose the mode

- **Prepare a release:** inspect changes, recommend a version, make requested version updates, validate, and draft notes. Do not push tags or create a published GitHub release. Present the concrete version, commit, notes, and validation for review.
- **Publish a release:** explicit instructions to release/publish/tag a version authorize the relevant publishing actions. Complete preparation, then publish without asking for the same authorization again. If the version is unspecified, recommend one from the changes and request the missing version decision while continuing preparation.
- Creating this skill, pushing a feature branch, or opening a PR is not a request to publish the project.

## Establish the release target

Run from the repository root:

```text
git status --short
git remote -v
git fetch origin --tags
git branch --show-current
git tag --list "v*" --sort=-version:refname
```

Verify the remote is the intended repository (currently `jsness/recipe-extractor`). Default the release target to fetched `origin/main`, not the current feature branch. Respect an explicitly requested branch or commit. Resolve and record the target SHA, and validate that exact code in a clean checkout; use an isolated worktree when the current checkout contains unrelated work. Do not stash, reset, stage, or include unrelated files.

Find the most recent release tag reachable from the target using `git describe --tags --match "v*" --abbrev=0 <target>`. Check GitHub release metadata as well; distinguish prereleases from stable releases. Inspect the actual diff and commits from the prior tag to the target. Do not choose a baseline solely by tag date or the largest version on another branch. If there is no prior reachable tag, review the target history as an initial release.

Use `vMAJOR.MINOR.PATCH`, matching existing tags. Recommend patch for fixes, minor for backward-compatible features, and major for breaking behavior or public Go/API contracts. Explain the choice briefly. Never silently change a user-specified version; flag an existing tag or a conflict with the intended changes.

## Prepare code and release notes

Search version references rather than blindly replacing numbers:

```text
rg -n 'recipe-extractor/|version|release|v[0-9]+\.[0-9]+\.[0-9]+' server web/package.json README.md docs .github
```

The scraper user agent in `server/scraper/scraper.go` currently contains the project version. Update it when preparing version changes, along with any genuine release-version references discovered. `web/package.json` is a private app with placeholder version `0.0.0`; do not bump it merely to match a Git tag. Keep any necessary lockfile updates consistent. The Go module path does not need changing for a routine v1 release.

Commit only release-specific changes when committing is within scope. If release preparation requires changes to main, prepare them on a release branch for review; merge only when authorized. After the changes land, refetch and resolve the intended target SHA again. A feature branch push does not put those changes into a main release.

Draft notes from the actual diff, describing user-visible features, fixes, configuration changes, database migration requirements, and breaking changes. Omit empty sections and internal commit noise. Do not invent compatibility guarantees or successful tests. Include the prior-tag comparison link when available. Save notes to a temporary UTF-8 file for CLI publication; use structured body arguments for an API tool.

## Validate the release candidate

Run on the candidate checkout, using directory-specific commands from `AGENTS.md`:

| Directory | Checks |
| --- | --- |
| `server/` | `go test ./...`, `go build ./...` |
| `web/` | `npm ci`, `npm run lint`, `npx tsc --noEmit`, `npm run build` |
| root | `git diff --check`, PowerShell `scripts/check-bom.ps1`, `docker compose build app` |

The Docker build verifies the UI is copied into the Go embedded frontend; a direct Go build alone does not. A local Docker build is not proof that both release architectures publish successfully. Do not launch or replace the user's running app to perform release validation.

Resolve failures before publishing. If a required check is unavailable, disclose the exact gap and finish the notes/preparation; request a decision on proceeding only when the unresolved gap prevents treating the candidate as validated. Never claim a check passed because CI is expected to run. The publishing workflow currently builds images but does not run the test suite.

Before tagging, verify the release target contains the intended version changes, all validation refers to that SHA, and no tracked changes remain in its validation checkout.

## Publish when authorized

Use the available GitHub connector, API, or authenticated CLI. Do not require `gh` if another available tool can create a release; if no authenticated publishing capability is available, preserve the prepared result and report that limitation.

1. Check the exact tag both locally and remotely (`git ls-remote --tags origin "refs/tags/<version>" "refs/tags/<version>^{}"`) and check for an existing GitHub release. An existing tag must resolve to the intended commit. Never force-move, delete, or overwrite a published tag.
2. For a new tag, create an annotated tag on the recorded SHA: `git tag -a <version> <sha> -m "Release <version>"`. Push only that tag with `git push origin refs/tags/<version>`; do not use `--tags` or force-push. Verify the remote tag resolves to the recorded commit.
3. Create the GitHub release using that existing tag, prepared title/notes, and the correct stable/prerelease setting. With `gh`, use `gh release create <version> --verify-tag --title <version> --notes-file <notes-path>` and `--prerelease` when appropriate. Do not upload binaries unless requested.
4. Inspect the tag-triggered `Publish Docker image` run for this exact tag/SHA. The current workflow publishes to `ghcr.io/jsness/recipe-extractor` for `linux/amd64` and `linux/arm64`; consult its metadata configuration and run output for emitted image tags. Do not assume `latest` moved simply because a release was created.
5. Verify the published image manifest/platforms where access permits. Report publication separately from workflow completion and image verification.

If a publishing call fails or returns uncertain results, read the remote tag/release state before retrying. Resume from the first incomplete step instead of recreating artifacts. If the tag pushed but release creation or Docker publishing failed, report the partial result and recovery action; leave the existing tag intact. Use bounded waits and report a pending workflow if it has not completed.

## Deliver

For preparation, provide the proposed version, target SHA, notes, changes made, and actual check results. For publication, provide the version, tag and GitHub release links, published SHA, Docker workflow status/link, and verified image tags/platforms or verification gaps.
