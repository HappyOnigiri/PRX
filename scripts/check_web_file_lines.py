#!/usr/bin/env python3
import subprocess
import sys
from pathlib import Path


WARNING_LINE_LIMIT = 600
HARD_LINE_LIMIT = 1000
TARGET_ROOTS = ("web/src", "web/tests")
EXCLUDED_ROOTS = (Path("web/src/gen"),)


def find_typescript_files(repository_root: Path) -> list[Path]:
    result = subprocess.run(
        [
            "git",
            "ls-files",
            "--cached",
            "--others",
            "--exclude-standard",
            "-z",
            "--",
            *TARGET_ROOTS,
        ],
        cwd=repository_root,
        check=True,
        stdout=subprocess.PIPE,
    )
    paths = (
        Path(raw_path.decode("utf-8"))
        for raw_path in result.stdout.split(b"\0")
        if raw_path
    )
    return sorted(
        path
        for path in paths
        if path.suffix in {".ts", ".tsx"}
        and not any(root == path or root in path.parents for root in EXCLUDED_ROOTS)
        and (repository_root / path).is_file()
    )


def main() -> int:
    repository_root = Path(__file__).resolve().parents[1]
    warnings: list[str] = []
    errors: list[str] = []

    for path in find_typescript_files(repository_root):
        try:
            line_count = len(
                (repository_root / path).read_text(encoding="utf-8").splitlines()
            )
        except (OSError, UnicodeError) as error:
            errors.append(f"{path}: failed to read file: {error}")
            continue
        if line_count > HARD_LINE_LIMIT:
            errors.append(
                f"{path}: {line_count} lines (maximum is {HARD_LINE_LIMIT})"
            )
        elif line_count > WARNING_LINE_LIMIT:
            warnings.append(
                f"{path}: {line_count} lines (warning above {WARNING_LINE_LIMIT})"
            )

    if warnings:
        print("Web TypeScript file-size warnings:")
        print("\n".join(f"- {warning}" for warning in warnings))
    if errors:
        print("Web TypeScript file-size errors:")
        print("\n".join(f"- {error}" for error in errors))
        return 1

    print(f"No Web TypeScript files exceed {HARD_LINE_LIMIT} lines.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
