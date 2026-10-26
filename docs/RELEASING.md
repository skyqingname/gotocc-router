# GoToCC release publication

## One release path

Adapt the latest Plus release locally, build the final Linux/arm64 and Linux/amd64 packages,
run the matching package in the retained local environment, and let the owner manually
accept it. After acceptance and a publication request, upload **those same
packages** to `skyqingname/gotocc-router` GitHub Releases. The owner then clicks
update in the online owned-version panel and restarts when prompted.

The containing Oracle maintenance workspace owns `scripts/package_plus.py`,
parameters in `release-parameters.json` and `sub2api-plus-workflow.json`, and
the procedures in `docs/maintenance-update-rules.md` and `docs/plus-maintenance.md`.
Those paths belong to that workspace, not this product checkout. The older
`scripts/sub2api-plus-upgrade.sh` is a status/reference entry whose historical
build/publish actions are not the current dual-architecture release path. The product version is in
`backend/cmd/server/VERSION`; upstream identity is in `UPSTREAM.md`.

## Before acceptance

Complete and verify the feature first. Assign the owned release version only
after the owner confirms the release arrangement; acceptance for a later combined
release does not authorize an independent PR or version. Upstream baseline
metadata is updated during adaptation and does not wait for that assignment.

Complete the generated runtime assets, version and release notes under
[Generated artifacts](GENERATED_ARTIFACTS.md), then fix the source commit.
Changing `release-channel.json` requires regenerating its Go constants and
reviewing the existing generator's full output before packaging. Current build
entries do not do this automatically. Build the package from that commit. Run its embedded frontend,
backend and matching external resources in the local acceptance environment.
Fix concrete faults and verify affected behavior. Keep the local environment
available for the owner. No full test matrix or four-level promotion gate is
required for this locally accepted release path. Under the owner's execution
rules, use builds, type/static checks, existing generator checks and actual
behavior verification; do not add or run test suites or local SHA sums.

Verify the final running package's owned version and upstream baseline through
the version panels or read-only update-check response. Check the upstream warning
and update indication against the actual fetched release. `-version`, packaged
JSON and successful downloads alone do not prove this. Metadata changed after
feature acceptance still needs affected generation and package verification;
unchanged feature behavior does not need another full matrix.

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
The pricing URL must contain the final immutable `/releases/download/<tag>/`
path. Draft upload responses can expose an `untagged-*` URL; never copy that
transient URL into the pricing manifest. The upstream `.003` validator rejects
it even after publication. Verify this metadata before publishing the draft.

## After acceptance

1. Push the accepted source branch, open and merge its PR to main, then tag the
   fixed source commit recorded in both packages. Verify the owned remote and
   peeled commit: an upstream tag may have the same name. Push that immutable tag.
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

`release-channel.json` is the authoritative release-channel input; generated Go
constants carry it into the updater binary. Updating only the packaged JSON
does not change the running updater's compiled baseline.
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
