"""Validate declared AI attribution in commits; unmarked human commits are allowed."""

import re
import subprocess
import sys


def validate_message(message):
    parsed = subprocess.run(
        ["git", "interpret-trailers", "--parse"],
        input=message, text=True, capture_output=True, check=True,
    ).stdout
    fields = {}
    for line in parsed.splitlines():
        key, separator, value = line.partition(":")
        if separator and key.lower().startswith("ai-"):
            fields.setdefault(key.lower(), []).append(value.strip())
    declared = re.search(r"^AI-(Agent|Model|On-Behalf-Of)\s*:", message, re.M | re.I)
    if not declared:
        return []
    errors = []
    for key in ("ai-agent", "ai-model", "ai-on-behalf-of"):
        values = fields.get(key, [])
        if len(values) != 1 or not values[0]:
            errors.append(f"{key} must appear once with a value in the trailer block")
    model = fields.get("ai-model", [])
    if len(model) == 1 and not re.fullmatch(
        r"[A-Za-z0-9._:/-]+(,[A-Za-z0-9._:/-]+)*", model[0]
    ):
        errors.append("ai-model must be a model identifier, unknown, or comma-separated identifiers")
    return errors


def main():
    if len(sys.argv) != 2:
        print("Usage: python scripts/check-ai-attribution.py <commit-or-range>")
        return 2
    commits = subprocess.check_output(
        ["git", "rev-list", sys.argv[1]], text=True,
    ).splitlines()
    failed = False
    for commit in commits:
        message = subprocess.check_output(
            ["git", "show", "-s", "--format=%B", commit], text=True,
        )
        for error in validate_message(message):
            print(f"{commit[:12]}: {error}")
            failed = True
    if not failed:
        print(f"Attribution checks passed for {len(commits)} commits.")
    return int(failed)


if __name__ == "__main__":
    sys.exit(main())
