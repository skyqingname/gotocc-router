# GoToCC release publication

## One release path

Adapt the latest Plus release locally, build the final Linux/arm64 and Linux/amd64 packages,
run the matching package in the retained local environment, and let the owner manually
accept it. After acceptance and a publication request, upload **that same
packages** to `skyqingname/gotocc-router` GitHub Releases. The owner then clicks
update in the online owned-version panel and restarts when prompted.

The operations workspace owns the commands in `scripts/sub2api-plus-upgrade.sh`,
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

Every release must include real Linux/arm64 and Linux/amd64 archives from the
same fixed source commit and embedded frontend. Oracle uses the ARM64 archive. Record the target architecture independently from the
builder architecture. The package contains the `sub2api` binary,
`release-channel.json`, bundled pricing data and existing distribution documents.
Prepare these five GitHub assets before publishing the draft:

- `sub2api_<version>_linux_arm64.tar.gz`
- `sub2api_<version>_linux_amd64.tar.gz`
- `checksums.txt` required by the installed updater
- `model-pricing.json`
- `model-pricing-manifest.json`

The existing updater manifest format is unchanged. Under the owner's no-SHA-sum
instruction, use the digest returned by GitHub for each uploaded archive to write
its filename-specific checksum line while the release is still a draft. Do not calculate a new
local checksum or reuse an amd64 checksum for an ARM64 archive. Unchanged pricing
bytes retain their existing manifest digest; update the version and download URL.

## After acceptance

1. Push the accepted source branch, open and merge its PR to main, then tag the
   fixed source commit recorded in both packages. Push that immutable tag.
2. Create a draft Release and upload both archives. Use their GitHub asset digests
   for the two checksum lines, then upload the checksum and pricing assets.
3. Publish the complete draft as latest. Never publish an empty immutable
   Release and try to attach missing assets afterward.

Do not rebuild, repackage, run another application matrix, wait for GitHub
builds, or require preparation/finalization PRs. The source PR must describe the actual acceptance evidence. It does not
change the accepted tag or require a replacement binary. Published tags are
never reused or moved. A changed deliverable needs renewed acceptance of the
changed behavior before publication.

If an upload fails, retain the local package and draft and resume asset upload.
Do not manufacture another build just to retry publication.

## Online update

`release-channel.json` directs installs and rollbacks to the owned repository.
The reference Plus repository is only an update notice. An ordinary source
commit alone is not an installable release; the assets must be published.

Release notes state actual changes, upstream baseline and any migration,
configuration or rollback limitations. Publishing is not deployment. Update
the operations production lock only after the online update is confirmed.
The installed updater constructs `sub2api_<version>_<GOOS>_<GOARCH>.tar.gz`
and matches the complete asset name, then finds that same name in `checksums.txt`.
Both architectures can coexist in one Release without changing the updater.

Routine updates do not require another manual SSH deployment. Preserve data
and backups; replacing an old binary does not reverse a forward migration.
