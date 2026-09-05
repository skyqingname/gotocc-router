# Release Process

The default is local validation and local artifact publication. GitHub hosts the
source, pull requests and immutable Release assets; a GitHub build or repeated
merged-main CI run is not required. The repository Release workflow is disabled
and has no automatic tag trigger. Ordinary CI may run as background feedback.

## Version Format

| Surface | Format |
| --- | --- |
| Git tag and GitHub Release | `vX.Y.Z+custom.NNN` |
| Embedded application version | `X.Y.Z+custom.NNN` |
| OCI image tag | `vX.Y.Z-custom.NNN` |

Increment `NNN` on the same upstream baseline and reset it to `001` after
merging a newer official baseline. `NNN` is always three digits.

## Console Release Indicators

The gray version badge opens the owned repository's release panel. Its update,
download and rollback actions use the owned release channel. The adjacent
upstream badge is gray when the adopted upstream release is current and amber
when a newer upstream release is available. Opening it refreshes release
metadata and shows the adopted baseline, latest upstream version and upstream
release/changelog link. Its refresh action fetches information only; installation
continues through a locally adapted owned release.

## Immutable Asset Publication

Create a draft, upload the Linux/amd64 program archive, `checksums.txt`,
`model-pricing.json` and `model-pricing-manifest.json`, then publish the complete
draft. Published assets and tags are immutable. An incomplete published version
must keep its original assets and receive a new version number.

## Default Local Publication

1. Keep the version, owned repository, upstream baseline, source commit, build
   target, output paths and validation image in the task's parameter JSON.
2. Run the existing full application matrix in the pinned local Docker runtime
   for application changes. Reuse that successful exact tree; do not repeat the
   matrix just to publish. Metadata/documentation-only follow-ups use relevant
   focused Docker checks.
3. Submit and merge a PR with a successful `sub2api/local-validation` status.
   `main` requires a PR, an up-to-date branch and that local status. GitHub CI
   and Security Scan are background feedback, not merge or release gates.
4. Verify the actual merged commit has the locally validated tree. Build the
   embedded frontend and `go build -tags=embed` inside the pinned Docker runtime
   with Linux/amd64, CGO disabled, `main.Commit` set to that merge commit,
   `main.Date` to the build time and `main.BuildType=release`.
5. Package `sub2api`, `release-channel.json` and
   `resources/model-pricing/model_prices_and_context_window.json` at those exact
   archive paths. Use the name `sub2api_<version>_linux_amd64.tar.gz`. Generate the
   existing updater's archive checksum file and both standalone pricing assets;
   `go -C backend run ./cmd/pricing-manifest-build` creates the manifest from
   `--data`, `--repository`, `--tag` and `--manifest` arguments.
6. Create an annotated tag at the actual merge commit with
   `git tag -a --cleanup=verbatim -F <notes-file> <tag> <merge-commit>`, and push
   only `refs/tags/<tag>`. Keep the GitHub Release workflow disabled.
7. Use `gh release create <tag> --verify-tag --draft` with the reviewed notes and
   all four assets. Inspect the completed draft, then use
   `gh release edit <tag> --draft=false --latest`.
8. Download the published archive, compare its identity and pricing resources,
   and run that exact binary against the retained local acceptance environment.
   Deploy only after local runtime/API/browser verification. Production needs
   its own backup, upgrade and runtime acceptance; publication is not deployment.
9. Mark the version `published` in `UPSTREAM.md` through a separate locally
   checked PR. Preserve incomplete versions as `invalid` or `withdrawn`.

Only platforms and images actually built are listed in Release notes. A
Linux/amd64 binary publication must not claim that new OCI images or other
platform archives exist. Existing channel and pricing overrides keep priority.

## Legacy GitHub Actions Mode (Explicit Opt-in)

The remainder describes the retained Actions-dependent CLI. It is not the
default and must not be used to impose remote waits on local publication.
Re-enabling the disabled workflow or restoring its remote required checks needs
an explicit request to use this mode.

## Repository Prerequisites

Before automatic PR promotion is enabled, repository administrators must:

1. Enable GitHub repository Auto-merge and merge-commit mode.
2. Protect `main` with a ruleset that requires pull requests.
3. Require the branch to be current with `main` and require these exact status
   contexts: `sub2api/local-validation`, `deployment-config`, `test`,
   `frontend`, `golangci-lint`, `goreleaser-config`, `repository-policy`,
   `backend-security`, and `frontend-security`.
4. Keep those contexts synchronized with the CI and Security Scan job IDs so a
   renamed or later unvalidated job cannot silently leave the merge gate.
5. Block force pushes and branch deletion; do not give release-cli an admin
   bypass.
6. Configure the Actions environment named `release` with administrator bypass
   disabled, no required reviewer, wait timer, or custom gate, and exactly one
   deployment policy for tags matching `v*+custom.*`.
7. Add an active repository Tag ruleset for `refs/tags/v*+custom.*`. Do not add
   bypass actors or a creation restriction; block tag updates and deletion.

The source tree cannot create these external governance settings safely.
`release-cli promote-pr` verifies Auto-merge, merge-commit mode, the required
pull-request rule, strict current-branch policy, and the complete context list.
Before tag transfer, `release-cli publish` verifies the complete Environment
and Tag ruleset policy. Both actions fail closed when an external prerequisite
is absent.

## Prepare the Release PR

Start a working branch from the latest `origin/main`, then:

1. Confirm the official upstream tag and commit.
2. Update `backend/cmd/server/VERSION`.
3. Update every `ARG VERSION=` in `Dockerfile` and `backend/Dockerfile`.
4. Add the custom version to `UPSTREAM.md` with status `planned`.
5. Synchronize install, rollback, and image examples:
   `python3 tools/update_release_docs.py`.
6. Write `release-notes.md` from the template below.
7. Commit all repository changes. Never commit on or push directly to `main`.

Intermediate pushes are intentionally fast and skip local validation:

```bash
python3 skills/push-cli/scripts/push_cli.py push
```

When the branch is the final merge candidate, submit it once through the local
promotion gate:

```bash
python3 skills/push-cli/scripts/push_cli.py submit-pr
```

`submit-pr` defaults to `profile=full`: it fetches the current `origin/main`,
requires it in the branch, records exact base/head SHAs, runs the complete
matrix in three bounded platform-container lanes, refetches and rechecks both
SHAs, pushes the exact head, publishes the typed
`sub2api/local-validation` status, and creates or reuses the PR. Any later head
or base change requires another `submit-pr`. `check --serial` is available only
for diagnosis and same-commit timing comparisons; it runs the same check set.

The documentation updater reads the current version from
`backend/cmd/server/VERSION`. Its rollback example uses the nearest lower
`published` entry in `UPSTREAM.md`; it skips `planned`, `historical`,
`withdrawn`, and `invalid`. Check without writing files with:

```bash
python3 tools/update_release_docs.py --check
```

## Release Notes

The first non-empty line is the annotated-tag subject. `Changed` and `Fixed`
are optional; every other section below is required and non-empty.

```markdown
Sub2API Plus vX.Y.Z+custom.NNN

## Highlights

Describe the primary user-visible changes.

## Changed

Optional details.

## Compatibility and migration

None.

## Known issues

None.

## Upstream baseline

Official release: vX.Y.Z
Official commit: <40-character commit>
```

## Promote the Release PR

Promotion requires the explicit PR number, intended tag, and reviewed notes:

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN \
  --pr <number> \
  --notes-file release-notes.md
```

The tool verifies the PR is open, non-draft, same-repository, and targets
`main`. Its machine marker and successful local-validation status must match
the current head and current `main` base exactly. It waits for GitHub required
checks, rechecks both SHAs, and enables native `--auto --merge` without admin
bypass.

After GitHub merges the PR, release-cli resolves the actual merge commit,
fetches `origin/main`, and waits for push-triggered `CI` and `Security Scan`
runs at that exact SHA. A successful PR check alone is insufficient because
the merge commit may differ from the PR head.

If protected review or another required condition is still pending, promotion
returns status 2. Complete the GitHub requirement and rerun the same command.

## Validate and Create the Tag

After PR promotion, run the focused release metadata gate against the merged
PR commit, then repeat it while creating the local annotated tag:

```bash
python3 skills/release-cli/scripts/release_cli.py validate \
  --tag vX.Y.Z+custom.NNN \
  --pr <number> \
  --notes-file release-notes.md

python3 skills/release-cli/scripts/release_cli.py tag \
  --tag vX.Y.Z+custom.NNN \
  --pr <number> \
  --notes-file release-notes.md
```

This gate does not repeat Go, frontend, lint, integration, or deployment
matrices. Those ran in `submit-pr`, PR Actions, and merged-main Actions. It
validates release metadata, notes, synchronized examples, exact tree identity,
and local/remote tag absence. `tag` targets the PR's actual merge commit and
preserves the notes verbatim. It never pushes.

Review the local tag, then explicitly publish only that tag:

```bash
git show --no-patch vX.Y.Z+custom.NNN
python3 skills/release-cli/scripts/release_cli.py publish \
  --tag vX.Y.Z+custom.NNN
```

`publish` first verifies the automatic `release` Environment and immutable
custom-tag ruleset, then returns after exact tag transfer. This explicit command
is the irreversible publication authorization point. Never use `git push
--tags`, reuse a version, retag, force push, or create the GitHub Release
manually.

## Monitor and Verify

Monitor publication separately. The remote annotated tag is the source of
truth, so this action can resume without relying on a local tag:

```bash
python3 skills/release-cli/scripts/release_cli.py monitor \
  --tag vX.Y.Z+custom.NNN
```

The Release workflow runs a focused provenance gate before publishing. It
requires the annotated tag to target a commit contained by `main`, validates
the tag notes and planned mapping, and requires successful push-triggered `CI`
and `Security Scan` runs for that exact main SHA. It does not rerun backend,
frontend, lint, integration, deployment, or security application matrices.
After provenance succeeds, `Build and publish` enters the checked `release`
Environment and starts automatically. If it unexpectedly waits, `monitor`
reports policy drift and the Actions URL; restore the Environment policy and
rerun `monitor`. The CLI never approves or bypasses a deployment.

After the workflow succeeds, verify the published state:

```bash
python3 skills/release-cli/scripts/release_cli.py verify \
  --tag vX.Y.Z+custom.NNN
```

Verification requires a successful workflow, a non-draft/non-prerelease GitHub
Release, and both immutable pricing assets:

- `model-pricing.json`
- `model-pricing-manifest.json`

The workflow accepts an existing pricing asset only when its bytes are
identical. Correct a bad asset with a new custom version, never by replacement
or retagging.

## Finalize Through a PR

After verification, finalize the published mapping:

```bash
python3 skills/release-cli/scripts/release_cli.py finalize \
  --tag vX.Y.Z+custom.NNN
```

`finalize` fetches the latest `origin/main`, creates deterministic branch
`release/finalize-X.Y.Z-custom.NNN`, and changes exactly one `UPSTREAM.md`
status from `planned` to `published`. It validates that historical mapping
independently from the current embedded version. If a newer release was already
prepared, it also synchronizes the generated rollback examples; otherwise only
`UPSTREAM.md` changes. It then calls `push-cli submit-pr --profile
release-finalization --tag <tag>` and never commits or pushes `main` directly.
Push-cli regenerates the complete expected tree from the recorded base and
verifies the published Release and immutable assets; it does not run the full
application container matrix.

After the follow-up PR Actions pass, promote it without release notes:

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN \
  --pr <finalization-pr-number>
```

The no-notes form is accepted only when the typed proof carries the matching
tag and deterministic finalization branch. Promotion independently regenerates
the tree and requires the Release, immutable assets, and `published` mapping.
Every required PR and merged-main context runs the same classifier before it
selects focused finalization validation; ambiguous or non-deterministic changes
fail closed.

## Pricing Assets

The manifest binds the release tag, fixed asset URL, and data SHA-256. Runtime
loading also validates dedicated HTTPS hosts, response sizes, JSON shape, and
version rollback. Release publication authority is the pricing trust boundary,
so tag-creation authority and Release/package permissions must remain limited
to trusted maintainers.

## Failed or Invalid Releases

- Never reuse or retag a published version.
- Record an externally visible bad release as `withdrawn` or `invalid` in
  `UPSTREAM.md` through a separate PR.
- Publish corrections under the next custom iteration.
- If tag push succeeded but local observation was interrupted, resume with
  `monitor`; do not rerun `publish`.
- If `Build and publish` unexpectedly waits, restore the checked Environment
  policy and rerun `monitor`; do not approve the drifted deployment through the
  CLI.
- Deleting an unpublished tag or artifact requires an explicit audit and
  maintainer decision.
