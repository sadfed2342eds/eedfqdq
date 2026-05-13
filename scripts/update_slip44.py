#!/usr/bin/env python3
"""
Pulls the canonical SLIP-0044 coin-type registry from satoshilabs/slips and
either:
  - splices it into `derivations.md` between the markers
        <!-- SLIP44-BEGIN: ... -->
        ```
        ...
        ```
        <!-- SLIP44-END -->
  - or just prints it to stdout (--print / --output -).

Usage:
    python update_slip44.py                 # update derivations.md next to script / CWD
    python update_slip44.py --print         # just print the code block, don't touch files
    python update_slip44.py --file path.md  # explicit target
    python update_slip44.py --check         # exit 1 if target file would change (CI)
    python update_slip44.py --dry-run       # print unified diff, do not write
    python update_slip44.py --source URL    # use a custom URL / local path (md or json)

Stdlib only, no third-party deps. Python 3.9+.
"""
from __future__ import annotations

import argparse
import difflib
import re
import sys
import urllib.request
from pathlib import Path
from typing import Iterable, Optional

DEFAULT_SOURCE = (
    "https://raw.githubusercontent.com/satoshilabs/slips/master/slip-0044.md"
)

# Mirrors used only when --source is DEFAULT_SOURCE and the primary fails.
MIRRORS = [
    DEFAULT_SOURCE,
    "https://cdn.jsdelivr.net/gh/satoshilabs/slips@master/slip-0044.md",
    "https://raw.githubusercontent.com/bitcoinjs/bip44-constants/master/constants.json",
]

BEGIN_MARK = "<!-- SLIP44-BEGIN:"
END_MARK = "<!-- SLIP44-END -->"

# Matches `0x80000000` or just `80000000`.
HEX_RE = re.compile(r"^(?:0x)?[0-9a-fA-F]+$")
# Strip markdown link `[Name](url)` -> `Name`, and images `![alt](url)` -> `alt`.
MD_LINK_RE = re.compile(r"!?\[([^\]]*)\]\([^)]*\)")


def fetch(source: str) -> str:
    """Fetch from URL or read from a local file. Try mirrors for the default URL."""
    if "://" not in source:
        return Path(source).read_text(encoding="utf-8")

    urls = MIRRORS if source == DEFAULT_SOURCE else [source]
    last_err: Optional[Exception] = None
    for url in urls:
        try:
            req = urllib.request.Request(
                url,
                headers={
                    "User-Agent": "slip44-updater/1.1 (+https://github.com)",
                    "Accept": "*/*",
                },
            )
            with urllib.request.urlopen(req, timeout=30) as resp:
                return resp.read().decode("utf-8", errors="replace")
        except Exception as err:
            last_err = err
            print(f"  ! {url} failed: {err}", file=sys.stderr)
    assert last_err is not None
    raise RuntimeError(f"all sources failed, last error: {last_err}")


def _clean(cell: str) -> str:
    """Strip markdown links / formatting / whitespace from a table cell."""
    s = MD_LINK_RE.sub(r"\1", cell).strip()
    # remove leading/trailing backticks, asterisks, underscores escapes
    s = s.strip("`*_ ")
    s = s.replace("\\_", "_").replace("\\|", "|")
    return s


def _parse_hardened(cell: str) -> Optional[int]:
    """Parse either `0x80000000` or a plain int (decimal or hex)."""
    c = cell.strip()
    if not c:
        return None
    try:
        if c.lower().startswith("0x"):
            return int(c, 16)
        return int(c)
    except ValueError:
        return None


def parse_slip44_markdown(md: str) -> list[tuple[int, str, str]]:
    """Robust parser: split on '|', find columns for coin_type / ticker / name.

    Upstream slip-0044.md currently has rows like:
        | 0x80000000 | 0 | BTC | [Bitcoin](https://bitcoin.org/) |
    But layout changes over time (4 or 5 columns, with/without hex, etc.).
    We look for a cell that parses as hardened (>= 0x80000000) and derive the
    coin_type from it; ticker/name are the next two non-empty cells after.
    """
    rows: list[tuple[int, str, str]] = []
    seen: set[int] = set()
    HARDENED = 0x80000000

    for raw in md.splitlines():
        line = raw.strip()
        if not line.startswith("|") or not line.endswith("|"):
            continue
        # split and drop the empty leading/trailing cells produced by outer pipes
        cells = [c.strip() for c in line.strip("|").split("|")]
        if len(cells) < 3:
            continue
        # skip table header / separator (`---`)
        if all(set(c) <= set("-: ") and c for c in cells):
            continue
        if any(c.lower() in {"coin type", "coin_type", "path component", "path", "symbol", "coin", "ticker"}
               for c in cells):
            continue

        coin_type: Optional[int] = None
        rest_idx = 0
        for i, c in enumerate(cells):
            v = _parse_hardened(c)
            if v is not None and v >= HARDENED:
                coin_type = v - HARDENED
                rest_idx = i + 1
                break
            # or a plain decimal path component (< HARDENED, but >= 0)
        if coin_type is None:
            # fall back: sometimes the hex is missing and the first cell is decimal.
            v = _parse_hardened(cells[0])
            if v is not None and 0 <= v < HARDENED:
                coin_type = v
                rest_idx = 1

        if coin_type is None or coin_type in seen:
            continue

        # collect remaining non-empty cells; skip repeated decimal index column
        tail = [_clean(c) for c in cells[rest_idx:]]
        # if the very next cell is the same decimal coin_type, drop it
        if tail and _parse_hardened(tail[0]) == coin_type:
            tail = tail[1:]
        tail = [t for t in tail if t]

        sym = tail[0] if len(tail) >= 1 else "-"
        name = tail[1] if len(tail) >= 2 else (tail[0] if tail else "-")
        # if tail had only the name, sym becomes "-"
        if len(tail) == 1:
            sym, name = "-", tail[0]

        seen.add(coin_type)
        rows.append((coin_type, sym or "-", name or "-"))

    rows.sort(key=lambda r: r[0])
    return rows


def parse_bip44_constants_json(raw: str) -> list[tuple[int, str, str]]:
    """Fallback: bitcoinjs/bip44-constants constants.json (array of [hardened, sym, name])."""
    import json

    data = json.loads(raw)
    rows: list[tuple[int, str, str]] = []
    seen: set[int] = set()
    for item in data:
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
    stripped = raw.lstrip()
    if stripped.startswith("[") or stripped.startswith("{"):
        return parse_bip44_constants_json(raw)
    return parse_slip44_markdown(raw)


def render_block(rows: Iterable[tuple[int, str, str]]) -> str:
    rows = list(rows)
    if not rows:
        raise RuntimeError("refusing to render empty SLIP-44 table (parse returned 0 rows)")

    ct_w = max(len(str(r[0])) for r in rows)
    sym_w = min(max(len(r[1]) for r in rows), 12)

    lines = ["```"]
    for ct, sym, name in rows:
        lines.append(f" {ct:<{ct_w}}  {sym[:sym_w]:<{sym_w}}  {name}")
    lines.append("```")
    return "\n".join(lines)


def splice(doc: str, new_block: str) -> str:
    begin_idx = doc.find(BEGIN_MARK)
    end_idx = doc.find(END_MARK)
    if begin_idx == -1 or end_idx == -1 or end_idx < begin_idx:
        raise RuntimeError(
            f"markers not found in target file — expected {BEGIN_MARK!r} ... {END_MARK!r}"
        )
    begin_line_end = doc.find("\n", begin_idx)
    if begin_line_end == -1:
        raise RuntimeError("malformed BEGIN marker")
    before = doc[: begin_line_end + 1]
    after = doc[end_idx:]
    return f"{before}{new_block}\n{after}"


def resolve_target(explicit: Optional[str]) -> Optional[Path]:
    """Find derivations.md unless --print/output was chosen.

    Search order:
      1. explicit --file argument
      2. ./derivations.md (current working directory)
      3. <script_dir>/derivations.md
      4. <script_dir>/../derivations.md (repo layout: scripts/ alongside the doc)
    """
    if explicit:
        return Path(explicit)
    script_dir = Path(__file__).resolve().parent
    candidates = [
        Path.cwd() / "derivations.md",
        script_dir / "derivations.md",
        script_dir.parent / "derivations.md",
    ]
    for p in candidates:
        if p.is_file():
            return p
    return None


def _force_utf8_stdio() -> None:
    """Windows consoles default to cp1251 / cp866 and choke on coin names like
    'Monero', 'Zcash' with accented chars. Reconfigure std streams to UTF-8.
    Silently ignored on older Python / non-reconfigurable streams."""
    for stream_name in ("stdout", "stderr"):
        stream = getattr(sys, stream_name, None)
        reconfigure = getattr(stream, "reconfigure", None)
        if reconfigure is not None:
            try:
                reconfigure(encoding="utf-8", errors="replace")
            except Exception:
                pass


def main() -> int:
    _force_utf8_stdio()
    ap = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    ap.add_argument("--source", default=DEFAULT_SOURCE,
                    help="URL or local file (markdown or bip44-constants.json)")
    ap.add_argument("--file", default=None,
                    help="path to derivations.md (default: search CWD / script dir)")
    ap.add_argument("--output", default=None,
                    help="write result to this path instead of patching --file; '-' = stdout")
    ap.add_argument("--print", dest="print_only", action="store_true",
                    help="just print the generated code block to stdout, touch no files")
    ap.add_argument("--check", action="store_true",
                    help="exit 1 if --file would change (for CI)")
    ap.add_argument("--dry-run", action="store_true",
                    help="print unified diff of --file, do not write")
    args = ap.parse_args()

    print(f"[*] fetching {args.source}", file=sys.stderr)
    raw = fetch(args.source)
    print(f"[*] parsing ({len(raw)} bytes)", file=sys.stderr)
    rows = parse_any(raw)
    print(f"[*] {len(rows)} coin types parsed", file=sys.stderr)

    block = render_block(rows)

    # Standalone mode: just print (or write a plain file), never touch derivations.md.
    if args.print_only or args.output == "-":
        sys.stdout.write(block + "\n")
        return 0
    if args.output:
        Path(args.output).write_text(block + "\n", encoding="utf-8")
        print(f"[+] wrote {args.output} ({len(rows)} coin types)", file=sys.stderr)
        return 0

    # Splice mode: find derivations.md.
    target = resolve_target(args.file)
    if target is None:
        print(
            "[!] derivations.md not found.\n"
            "    Pass --file PATH, or use --print to just print the block,\n"
            "    or put the script next to derivations.md.",
            file=sys.stderr,
        )
        return 2
    if not target.is_file():
        print(f"[!] {target} does not exist. Use --print or --output to skip splicing.",
              file=sys.stderr)
        return 2

    old = target.read_text(encoding="utf-8")
    try:
        new = splice(old, block)
    except RuntimeError as e:
        print(f"[!] {e}", file=sys.stderr)
        print("    Tip: add these two marker lines around a code block in the target file:\n"
              f"         {BEGIN_MARK} auto-generated -->\n"
              "         ```\n"
              "         (content managed by the script)\n"
              "         ```\n"
              f"         {END_MARK}",
              file=sys.stderr)
        return 3

    if old == new:
        print(f"[=] {target} already up-to-date", file=sys.stderr)
        return 0

    if args.check:
        print(f"[!] {target} is out of date (run without --check to update)", file=sys.stderr)
        sys.stdout.writelines(
            difflib.unified_diff(
                old.splitlines(keepends=True),
                new.splitlines(keepends=True),
                fromfile=f"a/{target.name}", tofile=f"b/{target.name}",
            )
        )
        return 1

    if args.dry_run:
        sys.stdout.writelines(
            difflib.unified_diff(
                old.splitlines(keepends=True),
                new.splitlines(keepends=True),
                fromfile=f"a/{target.name}", tofile=f"b/{target.name}",
            )
        )
        return 0

    target.write_text(new, encoding="utf-8")
    print(f"[+] wrote {target} ({len(rows)} coin types)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
