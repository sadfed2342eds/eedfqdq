#!/usr/bin/env python3
"""
Re-sorts the derivation-path list by liquidity / market-cap rank.

Default ranking is a curated top-150 snapshot (late 2025 / early 2026) of
the most-liquid crypto assets for which a SLIP-0044 coin_type exists.
Everything not in the top-150 is appended after, sorted by coin_type
(that preserves a stable order without implying a fake ranking).

Optionally accepts an external CSV (`ticker,name_hint,rank`) to override
the built-in ranking — e.g. pulled from CoinGecko / CoinMarketCap.

Usage:
    python sort_by_liquidity.py --source ../slip-0044.md --output sorted.md
    python sort_by_liquidity.py --ranking-csv my_ranks.csv --output sorted.md
    python sort_by_liquidity.py --top 150 --output top150.md
"""
from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from update_slip44 import DEFAULT_SOURCE, fetch, parse_any, _force_utf8_stdio  # noqa: E402
from generate_paths import path_for  # noqa: E402


# -- Built-in ranking ---------------------------------------------------------
# Curated "top liquid assets that live on their own chain" (late 2025/2026).
# Stable-coins that are pure tokens on existing chains (USDT/USDC/DAI, etc.)
# are NOT in this list — they have no SLIP-44 of their own; their liquidity
# attaches to whichever chain they run on (ETH=60, TRX=195, SOL=501, ...).
RANKING: list[tuple[str, str]] = [
    # (ticker, disambiguation substring; empty = any match)
    # Tier S
    ("BTC", ""),
    ("ETH", ""),
    # Tier A
    ("XRP", ""),
    ("BNB", ""),
    ("SOL", ""),
    ("DOGE", ""),
    ("TRX", ""),
    ("ADA", ""),
    ("TON", ""),
    ("AVAX", ""),
    ("SHIB", ""),
    ("LINK", ""),
    ("DOT", ""),
    ("BCH", ""),
    ("LTC", ""),
    ("XLM", ""),
    ("NEAR", ""),
    ("ICP", ""),
    ("APT", "aptos"),
    ("APTOS", ""),
    ("SUI", ""),
    ("HBAR", ""),
    ("UNI", ""),
    ("ETC", ""),
    ("KAS", ""),
    ("LEO", ""),
    ("XMR", ""),
    ("FIL", ""),
    # Tier B
    ("ATOM", ""),
    ("CRO", ""),
    ("MNT", ""),
    ("VET", ""),
    ("TIA", ""),
    ("ALGO", ""),
    ("RUNE", ""),
    ("INJ", ""),
    ("IMX", ""),
    ("ARB", ""),
    ("OP", ""),
    ("MATIC", ""),
    ("POL", ""),
    ("STX", ""),
    ("FET", ""),
    ("TAO", ""),
    ("AAVE", ""),
    ("MKR", ""),
    ("ZEC", ""),
    ("FLR", ""),
    ("THETA", ""),
    ("EGLD", ""),
    ("XTZ", ""),
    ("QNT", ""),
    ("CHZ", ""),
    ("NEO", ""),
    ("ONE", ""),
    ("KDA", ""),
    ("BAT", ""),
    ("SNX", ""),
    ("PAXG", ""),
    ("ROSE", ""),
    ("AXS", ""),
    ("SAND", ""),
    ("MANA", ""),
    ("MINA", ""),
    ("DASH", ""),
    ("IOTA", ""),
    ("SMR", ""),
    ("EOS", ""),
    ("ZRX", ""),
    ("ONT", ""),
    ("ICX", ""),
    ("KAVA", ""),
    ("ENS", ""),
    ("ENJ", ""),
    ("LDO", ""),
    ("CRV", ""),
    ("WAVES", ""),
    ("RVN", ""),
    ("BSV", ""),
    ("BTG", ""),
    ("GNO", ""),
    ("QTUM", ""),
    ("LRC", ""),
    ("COMP", ""),
    ("DCR", ""),
    ("XNO", ""),
    ("SEI", ""),
    ("FLOW", ""),
    ("KLAY", ""),
    ("KSM", ""),
    ("CKB", ""),
    ("AR", ""),
    ("WLD", ""),
    ("STRK", ""),
    ("WBT", ""),
    # Tier C
    ("TRB", ""),
    ("PEPE", ""),
    ("FLOKI", ""),
    ("GRT", ""),
    ("ONDO", ""),
    ("ENA", ""),
    ("ANT", ""),
    ("BAND", ""),
    ("BAL", ""),
    ("CELO", ""),
    ("HNT", ""),
    ("IOTX", ""),
    ("ANKR", ""),
    ("YFI", ""),
    ("ZEN", ""),
    ("GLM", ""),
    ("RSR", ""),
    ("STORJ", ""),
    ("CTSI", ""),
    ("OCEAN", ""),
    ("1INCH", ""),
    ("SUSHI", ""),
    ("CVX", ""),
    ("GMX", ""),
    ("SKL", ""),
    ("NKN", ""),
    ("SXP", ""),
    ("REQ", ""),
    ("CELR", ""),
    ("AUDIO", ""),
    ("SC", ""),
    ("DGB", ""),
    ("ZIL", ""),
    ("LSK", ""),
    ("XVG", ""),
    ("SYS", ""),
    ("FIRO", ""),
    ("BTM", ""),
    ("KMD", ""),
    ("PART", ""),
    ("PIVX", ""),
    ("VTC", ""),
    ("MONA", ""),
    ("GRS", ""),
    ("ARK", ""),
    ("BNT", ""),
    ("NMR", ""),
    ("REP", ""),
    ("POWR", ""),
    ("MTL", ""),
    ("POA", ""),
    ("CLO", ""),
    ("XDC", ""),
    ("TOMO", ""),
    ("WAN", ""),
    ("UBQ", ""),
    ("ETHO", ""),
    ("MUSIC", ""),
    ("ELLA", ""),
    ("EXP", ""),
    ("BAN", ""),
    ("HNS", ""),
    ("BEAM", ""),
    ("GRIN", ""),
    ("LBC", ""),
    ("GAME", ""),
    ("STEEM", ""),
    ("HIVE", ""),
    ("NEM", ""),
    ("XEM", ""),
    ("RBTC", ""),
]


def match_rank(rows: list[tuple[int, str, str]]) -> dict[int, int]:
    """Map coin_type -> rank position (1-based). Unranked coins omitted."""
    rank_by_ct: dict[int, int] = {}
    by_ticker: dict[str, list[tuple[int, str, str]]] = {}
    for ct, sym, name in rows:
        by_ticker.setdefault(sym.upper(), []).append((ct, sym, name))

    for pos, (ticker, disamb) in enumerate(RANKING, start=1):
        candidates = by_ticker.get(ticker.upper(), [])
        if not candidates:
            continue
        if disamb:
            disamb_l = disamb.lower()
            picked = next(
                (c for c in candidates if disamb_l in c[2].lower()),
                candidates[0],
            )
        else:
            picked = candidates[0]
        ct = picked[0]
        if ct not in rank_by_ct or rank_by_ct[ct] > pos:
            rank_by_ct[ct] = pos
    return rank_by_ct


def load_csv_ranks(path: Path) -> list[tuple[str, str]]:
    entries: list[tuple[str, str, int]] = []
    with path.open(encoding="utf-8") as f:
        reader = csv.reader(f)
        header_skipped = False
        for row in reader:
            if not row:
                continue
            if not header_skipped:
                header_skipped = True
                try:
                    int(row[-1])
                except ValueError:
                    continue
            try:
                rank = int(row[-1])
            except ValueError:
                continue
            ticker = row[0].strip()
            name_hint = row[1].strip() if len(row) >= 3 else ""
            entries.append((ticker, name_hint, rank))
    entries.sort(key=lambda e: e[2])
    return [(t, h) for t, h, _ in entries]


def main() -> int:
    _force_utf8_stdio()
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--source", default=DEFAULT_SOURCE)
    ap.add_argument("--ranking-csv", default=None,
                    help="Optional CSV 'ticker,name_hint,rank' to override built-in")
    ap.add_argument("--output", default="-")
    ap.add_argument("--top", type=int, default=0,
                    help="Emit only the top-N ranked chains, drop the long tail")
    args = ap.parse_args()

    print(f"[*] fetching {args.source}", file=sys.stderr)
    raw = fetch(args.source)
    rows = parse_any(raw)
    print(f"[*] {len(rows)} coin types parsed", file=sys.stderr)

    if args.ranking_csv:
        global RANKING
        RANKING = load_csv_ranks(Path(args.ranking_csv))
        print(f"[*] loaded {len(RANKING)} tickers from {args.ranking_csv}", file=sys.stderr)

    rank_by_ct = match_rank(rows)
    print(f"[*] matched {len(rank_by_ct)} chains against ranking", file=sys.stderr)

    ranked = sorted(
        (r for r in rows if r[0] in rank_by_ct),
        key=lambda r: (rank_by_ct[r[0]], r[0]),
    )
    unranked = sorted(
        (r for r in rows if r[0] not in rank_by_ct),
        key=lambda r: r[0],
    )

    if args.top:
        ranked = ranked[: args.top]
        unranked = []

    lines = [
        "# Derivation paths sorted by liquidity (late-2025 / 2026 snapshot)",
        "",
        "Methodology: curated top-150 by CEX spot liquidity and market cap as of",
        "2025-2026. Stablecoins (USDT/USDC/DAI/BUSD/TUSD/FDUSD) are excluded —",
        "they are tokens on other chains (ETH=60, TRX=195, SOL=501, ...) and inherit",
        "that chain's path. Rows outside the top-150 are listed afterwards sorted",
        "by SLIP-44 coin_type (stable order, not a ranking).",
        "",
        "| rank | coin_type | ticker | name | derivation path |",
        "|-----:|----------:|--------|------|-----------------|",
    ]
    for r in ranked:
        ct, sym, name = r
        pos = rank_by_ct[ct]
        safe_name = name.replace("|", r"\|")
        safe_path = path_for(ct).replace("|", r"\|")
        lines.append(f"| {pos} | {ct} | {sym} | {safe_name} | `{safe_path}` |")

    if unranked and not args.top:
        lines.append("")
        lines.append(f"## Long tail — {len(unranked)} chains not in top-150")
        lines.append("")
        lines.append("Micro-caps, dormant / delisted projects, unissued reservations, or chains")
        lines.append("whose tickers we cannot match confidently. Listed in coin_type order.")
        lines.append("")
        lines.append("| coin_type | ticker | name | derivation path |")
        lines.append("|----------:|--------|------|-----------------|")
        for ct, sym, name in unranked:
            safe_name = name.replace("|", r"\|")
            safe_path = path_for(ct).replace("|", r"\|")
            lines.append(f"| {ct} | {sym} | {safe_name} | `{safe_path}` |")

    text = "\n".join(lines) + "\n"
    if args.output == "-":
        sys.stdout.write(text)
    else:
        Path(args.output).write_text(text, encoding="utf-8")
        print(f"[+] wrote {args.output}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
