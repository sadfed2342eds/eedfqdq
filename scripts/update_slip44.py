#!/usr/bin/env python3
"""
Pulls the canonical SLIP-0044 coin-type registry from satoshilabs/slips and
splices it into `derivations.md` between the markers:

    <!-- SLIP44-BEGIN: ... -->
    ```
    ...
    ```
    <!-- SLIP44-END -->

Usage:
    python3 scripts/update_slip44.py                 # update derivations.md in-place
    python3 scripts/update_slip44.py --check         # exit 1 if file would change (CI)
    python3 scripts/update_slip44.py --dry-run       # print diff, do not write
    python3 scripts/update_slip44.py --source URL    # use a custom URL / local path

Stdlib only, no third-party deps.
"""
from __future__ import annotations

import argparse
import difflib
import re
import sys
import urllib.request
from pathlib import Path
from typing import Iterable

# Canonical source. Use the raw markdown so parsing stays stable.
DEFAULT_SOURCE = (
    "https://raw.githubusercontent.com/satoshilabs/slips/master/slip-0044.md"
)

# Mirrors if GitHub raw is unreachable (INTEGRATIONS_ONLY sandboxes, offline CI, ...).
MIRRORS = [
    DEFAULT_SOURCE,
    "https://cdn.jsdelivr.net/gh/satoshilabs/slips@master/slip-0044.md",
    "https://raw.githubusercontent.com/bitcoinjs/bip44-constants/master/constants.json",
]

BEGIN_MARK = "<!-- SLIP44-BEGIN:"
END_MARK = "<!-- SLIP44-END -->"

# Matches a single SLIP-0044 markdown table row. Example:
#     | 0x80000000 | 0  | BTC    | [Bitcoin](...)     |
# Columns: hex coin_type | dec | ticker (optional) | name (+ optional link/md).
ROW_RE = re.compile(
    r"""^\|\s*
        (?:0x[0-9a-fA-F]+)\s*\|\s*     # hex form (ignored, dec is authoritative)
        (?P<dec>\d+)\s*\|\s*           # decimal coin_type
        (?P<sym>[^|]*?)\s*\|\s*        # ticker / symbol (may be empty or reserved)
        (?P<name>[^|]*?)\s*\|\s*$      # coin name (may contain [name](url))
    """,
    re.VERBOSE,
)

# Strip markdown link syntax `[Name](url)` -> `Name`.
MD_LINK_RE = re.compile(r"\[([^\]]+)\]\([^)]*\)")


def fetch(source: str) -> str:
    """Fetch from URL or read from a local path. Try mirrors for the default URL."""
    if "://" not in source:
        return Path(source).read_text(encoding="utf-8")

    urls = MIRRORS if source == DEFAULT_SOURCE else [source]
    last_err: Exception | None = None
    for url in urls:
        try:
            req = urllib.request.Request(
                url,
                headers={"User-Agent": "slip44-updater/1.0 (+https://github.com)"},
            )
            with urllib.request.urlopen(req, timeout=30) as resp:
                return resp.read().decode("utf-8")
        except Exception as err:  # network, 403, 404, DNS, ...
            last_err = err
            print(f"  ! {url} failed: {err}", file=sys.stderr)
    assert last_err is not None
    raise RuntimeError(f"all sources failed, last error: {last_err}")


def parse_slip44_markdown(md: str) -> list[tuple[int, str, str]]:
    """Parse the SLIP-0044 markdown into (coin_type, ticker, name) tuples."""
    rows: list[tuple[int, str, str]] = []
    seen: set[int] = set()
    for line in md.splitlines():
        if not line.startswith("|"):
            continue
        m = ROW_RE.match(line)
        if not m:
            continue
        dec = int(m["dec"])
        if dec in seen:
            continue
        seen.add(dec)
        sym = MD_LINK_RE.sub(r"\1", m["sym"]).strip() or "-"
        name = MD_LINK_RE.sub(r"\1", m["name"]).strip() or "-"
        # markdown escapes like \_ -> _
        sym = sym.replace("\\_", "_")
        name = name.replace("\\_", "_")
        rows.append((dec, sym, name))
    rows.sort(key=lambda r: r[0])
    return rows


def parse_bip44_constants_json(raw: str) -> list[tuple[int, str, str]]:
    """Fallback parser for bitcoinjs/bip44-constants constants.json."""
    import json

    data = json.loads(raw)
    rows: list[tuple[int, str, str]] = []
    seen: set[int] = set()
    for item in data:
        # item = [0x80000000 + coin_type, "SYM", "Name"]
        try:
            hardened, sym, name = item[0], item[1], item[2]
        except (IndexError, TypeError):
            continue
        dec = int(hardened) - 0x80000000
        if dec < 0 or dec in seen:
            continue
        seen.add(dec)
        rows.append((dec, str(sym) or "-", str(name) or "-"))
    rows.sort(key=lambda r: r[0])
    return rows


def parse_any(raw: str) -> list[tuple[int, str, str]]:
    """Auto-detect format (markdown vs constants.json) and parse."""
    stripped = raw.lstrip()
    if stripped.startswith("[") or stripped.startswith("{"):
        return parse_bip44_constants_json(raw)
    return parse_slip44_markdown(raw)


def render_block(rows: Iterable[tuple[int, str, str]]) -> str:
    """Produce the fixed-width code block that goes between the markers."""
    rows = list(rows)
    if not rows:
        raise RuntimeError("refusing to render empty SLIP-44 table")

    ct_w = max(len(str(r[0])) for r in rows)
    sym_w = min(max(len(r[1]) for r in rows), 12)  # cap runaway tickers

    lines = ["```"]
    for ct, sym, name in rows:
        lines.append(f" {ct:<{ct_w}}  {sym[:sym_w]:<{sym_w}}  {name}")
    lines.append("```")
    return "\n".join(lines)


def splice(doc: str, new_block: str) -> str:
    """Replace the content between BEGIN/END markers. Markers themselves stay."""
    begin_idx = doc.find(BEGIN_MARK)
    end_idx = doc.find(END_MARK)
    if begin_idx == -1 or end_idx == -1 or end_idx < begin_idx:
        raise RuntimeError(
            f"markers not found in document — expected {BEGIN_MARK!r} ... {END_MARK!r}"
        )
    # Keep the BEGIN comment line intact; replace everything between it and END.
    begin_line_end = doc.find("\n", begin_idx)
    if begin_line_end == -1:
        raise RuntimeError("malformed BEGIN marker")
    before = doc[: begin_line_end + 1]
    after = doc[end_idx:]
    return f"{before}{new_block}\n{after}"


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--source", default=DEFAULT_SOURCE, help="URL or local file (md or json)")
    ap.add_argument("--file", default=str(Path(__file__).resolve().parent.parent / "derivations.md"),
                    help="path to derivations.md")
    ap.add_argument("--check", action="store_true", help="exit 1 if the file would change")
    ap.add_argument("--dry-run", action="store_true", help="print unified diff, do not write")
    args = ap.parse_args()

    print(f"[*] fetching {args.source}", file=sys.stderr)
    raw = fetch(args.source)
    print(f"[*] parsing ({len(raw)} bytes)", file=sys.stderr)
    rows = parse_any(raw)
    print(f"[*] {len(rows)} coin types parsed", file=sys.stderr)

    target = Path(args.file)
    old = target.read_text(encoding="utf-8")
    block = render_block(rows)
    new = splice(old, block)

    if old == new:
        print("[=] derivations.md already up-to-date", file=sys.stderr)
        return 0

    if args.check:
        print("[!] derivations.md is out of date (run without --check to update)", file=sys.stderr)
        sys.stdout.writelines(
            difflib.unified_diff(old.splitlines(keepends=True),
                                 new.splitlines(keepends=True),
                                 fromfile=f"a/{target.name}", tofile=f"b/{target.name}")
        )
        return 1

    if args.dry_run:
        sys.stdout.writelines(
            difflib.unified_diff(old.splitlines(keepends=True),
                                 new.splitlines(keepends=True),
                                 fromfile=f"a/{target.name}", tofile=f"b/{target.name}")
        )
        return 0

    target.write_text(new, encoding="utf-8")
    print(f"[+] wrote {target} ({len(rows)} coin types)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
