# GoToCC release publication

## One release path

Adapt the latest Plus release locally, build the final Linux/amd64 package,
run that package in the retained local environment, and let the owner manually
accept it. After acceptance and a publication request, upload **that same
package** to `skyqingname/gotocc-router` GitHub Releases. The owner then clicks
update in the online owned-version panel and restarts when prompted.

The vircs workspace owns the commands in `scripts/sub2api-plus-upgrade.sh`,
parameters in `sub2api-plus-workflow.json`, and the procedure in
`docs/sub2api-plus-upgrade-runbook.md`. Those paths belong to the containing
operations workspace, not this product checkout. The product version is in
`backend/cmd/server/VERSION`; upstream identity is in `UPSTREAM.md`.

## Before acceptance

Complete the code, generated runtime assets, version and release notes, then
commit locally. Build the package from that commit. Run its embedded frontend,
backend and matching external resources in the local acceptance environment.
Fix concrete faults and verify affected behavior. Keep the local environment
available for the owner. No full test matrix or four-level promotion gate is
required for this locally accepted release path.

The package contains the `sub2api` binary, `release-channel.json`, bundled
pricing data and the existing distribution documents. Prepare these four
GitHub assets during the local build:

- `sub2api_<version>_linux_amd64.tar.gz`
- `checksums.txt` required by the installed updater
- `model-pricing.json`
- `model-pricing-manifest.json`

The existing updater manifest format is unchanged. Generate it once during
packaging; do not add another hashing or download-validation stage.

## After acceptance

1. Tag the commit recorded in the accepted package and push the tag and source
   objects to the owned repository.
2. Create a draft Release and upload all four existing assets.
3. Publish the complete draft as latest. Never publish an empty immutable
   Release and try to attach missing assets afterward.

Do not rebuild, repackage, run another application matrix, wait for GitHub
builds, or require preparation/finalization PRs. PRs are optional source
collaboration and must describe the actual acceptance evidence. They do not
change the accepted tag or require a replacement binary. Published tags are
never reused or moved. A changed deliverable needs renewed acceptance of the
changed behavior before publication.

If an upload fails, retain the local package and draft and resume asset upload.
Do not manufacture another build just to retry publication. The legacy
`skills/push-cli` and `skills/release-cli` promotion workflows are outside this
path and should not be invoked unless the owner explicitly requests that mode.

## Online update

`release-channel.json` directs installs and rollbacks to the owned repository.
The reference Plus repository is only an update notice. An ordinary source
commit alone is not an installable release; the assets must be published.

Release notes state actual changes, upstream baseline and any migration,
configuration or rollback limitations. Publishing is not deployment. Update
the operations production lock only after the online update is confirmed.
Routine updates do not require another manual SSH deployment. Preserve data
and backups; replacing an old binary does not reverse a forward migration.
