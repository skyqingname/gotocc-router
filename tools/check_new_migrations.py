#!/usr/bin/env python3
"""Validate migration filenames and changes introduced since a Git base."""

from __future__ import annotations

import argparse
import hashlib
import re
import subprocess
import sys
from pathlib import Path

import release_docs


ROOT = Path(__file__).resolve().parents[1]
MIGRATIONS = ROOT / "backend/migrations"
NAME_RE = re.compile(r"^(\d{3})(?:[a-z])?_[a-z0-9_]+(?:_notx)?\.sql$")
ZERO_SHA_RE = re.compile(r"^0+$")

# These GotoCC migrations were already deployed before a later upstream release
# introduced files with the same numeric prefixes. Keep every exception exact
# and content-addressed: it permits only the released filename and bytes, never
# arbitrary reuse of an old numeric prefix.
REVIEWED_IMPORTED_MIGRATIONS = {
    "backend/migrations/220_reusable_invitation_codes.sql":
        "87b4518fd381c1009f19adcc02ced70af0f8a848e18f9ad0b15e7f6c2dcdb488",
    "backend/migrations/221_add_teams.sql":
        "eba1a35d52e7fb2cdff8f810240aea62ec04ce9d9b0b573a7be28985fe7434cb",
    "backend/migrations/222_harden_team_lifecycle.sql":
        "2fac2b538d5c98e64a69d283d06f2784e7e4d4084c4e1514df0b740333c16179",
    "backend/migrations/223_add_team_attribution_indexes_notx.sql":
        "5ec90906693a165116e3b2e845d7061036d46f969d90a9003b9f7f5c9837e75b",
    "backend/migrations/224_add_image_objects.sql":
        "bb077c8413a11a0f361b8629988857c2d4f7eb28092020735f8558321ffc6729",
    "backend/migrations/225_restore_openai_video_prices.sql":
        "bc69d7ff1c09eec409fe050af1b617d45e0a2017610a28c3cca790ba2da0cd13",
    "backend/migrations/238_openai_video_tasks.sql":
        "a4745fb45a29aae765d77cab027d4ffdd92d89755063532c8614c1ff571d1f2e",
    "backend/migrations/239_openai_video_task_billing_mode.sql":
        "3aa7672f9b178e970c79b3d6d586f9e6bde8ed9b6c98d42352a8d49111dc20b3",
}


def git(*args: str) -> str:
    return subprocess.check_output(
        ["git", *args], cwd=ROOT, text=True, encoding="utf-8"
    )


def validate_added(path: Path, errors: list[str]) -> int | None:
    match = NAME_RE.fullmatch(path.name)
    if not match:
        errors.append(
            f"{path.as_posix()}: expected NNN_description.sql or "
            "NNN_description_notx.sql"
        )
        return None

    content = path.read_text(encoding="utf-8")
    upper = content.upper()
    if path.name.endswith("_notx.sql"):
        if "CONCURRENTLY" not in upper:
            errors.append(f"{path.as_posix()}: _notx migration has no CONCURRENTLY")
        if re.search(r"\b(BEGIN|COMMIT|ROLLBACK)\b", upper):
            errors.append(f"{path.as_posix()}: _notx migration controls transactions")
        for statement in (part.strip() for part in content.split(";")):
            normalized = re.sub(r"--.*?$", "", statement, flags=re.MULTILINE).upper()
            if not normalized.strip():
                continue
            if "CONCURRENTLY" not in normalized:
                errors.append(
                    f"{path.as_posix()}: _notx migration mixes non-concurrent SQL"
                )
            if "CREATE" in normalized and "INDEX" in normalized:
                if "IF NOT EXISTS" not in normalized:
                    errors.append(
                        f"{path.as_posix()}: concurrent CREATE INDEX needs IF NOT EXISTS"
                    )
            elif "DROP" in normalized and "INDEX" in normalized:
                if "IF EXISTS" not in normalized:
                    errors.append(
                        f"{path.as_posix()}: concurrent DROP INDEX needs IF EXISTS"
                    )
            else:
                errors.append(
                    f"{path.as_posix()}: _notx only supports CREATE/DROP INDEX"
                )
    elif "CONCURRENTLY" in upper:
        errors.append(f"{path.as_posix()}: CONCURRENTLY requires _notx.sql")

    return int(match.group(1))


def is_reviewed_imported_migration(path: Path) -> bool:
    try:
        relative = path.relative_to(ROOT).as_posix()
    except ValueError:
        return False
    expected = REVIEWED_IMPORTED_MIGRATIONS.get(relative)
    if expected is None:
        return False
    return hashlib.sha256(path.read_bytes()).hexdigest() == expected


def has_duplicate_unreviewed_prefixes(paths: list[Path]) -> bool:
    prefixes = [
        int(match.group(1))
        for path in paths
        if not is_reviewed_imported_migration(path)
        if (match := NAME_RE.fullmatch(path.name))
    ]
    return len(prefixes) != len(set(prefixes))


def diff_changes(base: str) -> list[tuple[str, str]]:
    output = git(
        "diff",
        "--name-status",
        "--diff-filter=ADMRT",
        f"{base}...HEAD",
        "--",
        "backend/migrations",
    )
    changes: list[tuple[str, str]] = []
    for line in output.splitlines():
        fields = line.split("\t")
        if len(fields) >= 2:
            changes.append((fields[0], fields[-1]))
    return changes


def resolve_release_base(
    target: str,
    *,
    tags: list[str] | None = None,
    statuses: dict[str, str] | None = None,
) -> str:
    if release_docs.version_key(target) is None:
        raise ValueError(
            f"invalid target release {target!r}; expected vX.Y.Z+custom.NNN"
        )
    if statuses is None:
        statuses = release_docs.parse_upstream_statuses(
            ROOT.joinpath("UPSTREAM.md").read_text(encoding="utf-8")
        )
    if target not in statuses:
        raise ValueError(f"UPSTREAM.md has no mapping row for {target}")
    if tags is None:
        tags = git("tag", "--merged", "HEAD", "--list").splitlines()
    base = release_docs.select_previous_release_tag(tags, statuses, target)
    if base is None:
        raise ValueError(
            f"no earlier published or historical release tag is available for {target}"
        )
    return base


def main() -> int:
    parser = argparse.ArgumentParser()
    comparison = parser.add_mutually_exclusive_group()
    comparison.add_argument("--base", help="Git base SHA/ref for new-file validation")
    comparison.add_argument(
        "--target-release",
        help="release tag whose previous published/historical tag is the Git base",
    )
    args = parser.parse_args()
    errors: list[str] = []

    for path in sorted(MIGRATIONS.glob("*.sql")):
        if not NAME_RE.fullmatch(path.name):
            errors.append(f"{path.relative_to(ROOT).as_posix()}: invalid filename")

    base = (args.base or "").strip()
    if args.target_release:
        try:
            base = resolve_release_base(args.target_release)
        except (OSError, subprocess.CalledProcessError, ValueError) as error:
            print(f"Cannot resolve migration comparison base: {error}", file=sys.stderr)
            return 1
        print(f"Migration comparison base for {args.target_release}: {base}")
    if not base or ZERO_SHA_RE.fullmatch(base):
        if errors:
            for error in errors:
                print(f"- {error}", file=sys.stderr)
            return 1
        print("Migration filenames are valid; no Git base was provided.")
        return 0

    try:
        git("rev-parse", "--verify", f"{base}^{{commit}}")
        changes = diff_changes(base)
    except subprocess.CalledProcessError:
        print(f"Cannot resolve migration comparison base {base!r}.", file=sys.stderr)
        return 1

    added: list[Path] = []
    for status, relative in changes:
        if not relative.endswith(".sql"):
            continue
        if status == "A":
            added.append(ROOT / relative)
        else:
            errors.append(f"{relative}: existing migrations may not be {status}")

    base_files = git("ls-tree", "-r", "--name-only", base, "backend/migrations")
    base_prefixes = [
        int(match.group(1))
        for name in base_files.splitlines()
        if (match := NAME_RE.fullmatch(Path(name).name))
    ]
    max_base = max(base_prefixes, default=0)
    for path in added:
        prefix = validate_added(path, errors)
        if prefix is None:
            continue
        reviewed_import = is_reviewed_imported_migration(path)
        if prefix <= max_base and not reviewed_import:
            errors.append(
                f"{path.relative_to(ROOT).as_posix()}: prefix {prefix:03d} must be "
                f"greater than existing maximum {max_base:03d}"
            )
    if has_duplicate_unreviewed_prefixes(added):
        errors.append("new migrations contain duplicate numeric prefixes")

    if errors:
        print("Migration validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Migration changes are valid ({len(added)} new SQL file(s)).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
