#!/usr/bin/env python3
"""
Takes parsed SLIP-0044 rows and emits a single file listing a derivation path
template for EVERY coin type.

Rules:
  1. Default path for any coin_type C is  `m/44'/C'/0'/0/%d`  (BIP-44).
  2. A small hard-coded override table handles the ~30 chains whose wallets
     use a non-standard path (Cardano / Solana / Aptos / Sui / Stellar / TON
     / Polkadot / Hedera / MultiversX / Tezos / Algorand / NEAR / etc.).

Usage:
    python generate_paths.py                      # uses ../slip44_full.txt by default
    python generate_paths.py --source URL_OR_PATH --output paths.txt
    python generate_paths.py --markdown           # emit a markdown table instead of plain list
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

# Re-use the existing fetcher/parser so we don't re-implement it.
sys.path.insert(0, str(Path(__file__).resolve().parent))
from update_slip44 import DEFAULT_SOURCE, fetch, parse_any, _force_utf8_stdio  # noqa: E402

# Non-standard paths (canonical form, widely used by Phantom/Trust/Ledger Live
# and the chain's own spec). `%d` always marks the address-index slot.
# Multiple entries per coin = alternative paths, separated by " | ".
OVERRIDES: dict[int, str] = {
    # Cardano Shelley (payment key). Staking key is `m/1852'/1815'/0'/2/0`.
    1815: "m/1852'/1815'/%d'/0/0",
    # Solana — Phantom / Solflare default. Legacy Sollet: `m/44'/501'/%d'`.
    501: "m/44'/501'/%d'/0'",
    # Aptos / Sui — ed25519, every level hardened.
    637: "m/44'/637'/%d'/0'/0'",
    784: "m/44'/784'/%d'/0'/0'",
    # Stellar — ed25519, only one hardened index.
    148: "m/44'/148'/%d'",
    # NEAR — single-key; account index is via named account, not path.
    397: "m/44'/397'/0'",
    # TON — ed25519, single hardened segment.
    607: "m/44'/607'/%d'",
    # Tezos — ed25519 (tz1) default.
    1729: "m/44'/1729'/%d'/0'",
    # Hedera Hashgraph — ed25519, two levels hardened.
    3030: "m/44'/3030'/0'/0'/%d'",
    # MultiversX (Elrond).
    508: "m/44'/508'/0'/0'/%d'",
    # Algorand.
    283: "m/44'/283'/%d'/0'/0'",
    # Polkadot / Kusama (also support the `//polkadot//%d` BIP-39 passphrase form).
    354: "m/44'/354'/0'/0'/%d'",
    434: "m/44'/434'/0'/0'/%d'",
    # IOTA / Shimmer.
    4218: "m/44'/4218'/%d'/0'/0'",
    4219: "m/44'/4219'/%d'/0'/0'",
    # Mina (pallas).
    12586: "m/44'/12586'/%d'/0/0",
    # Radix (secp256k1, but canonical path).
    1022: "m/44'/1022'/0'/0/%d",
    # Filecoin (secp256k1, f1 address).
    461: "m/44'/461'/0'/0/%d",
    # Internet Computer (secp256k1).
    223: "m/44'/223'/0'/0/%d",
    # Bitcoin — native SegWit is the modern default, BIP-44 kept as fallback.
    0: "m/84'/0'/0'/0/%d  (BIP-84 SegWit) | m/44'/0'/0'/0/%d (legacy) | m/49'/0'/0'/0/%d (segwit-compat) | m/86'/0'/0'/0/%d (taproot)",
    # Litecoin — same story.
    2: "m/84'/2'/0'/0/%d  (BIP-84 SegWit) | m/44'/2'/0'/0/%d (legacy) | m/49'/2'/0'/0/%d (segwit-compat)",
    # Groestlcoin, Syscoin, Digibyte — SegWit-capable.
    17: "m/84'/17'/0'/0/%d",
    # Avalanche X/P-chain (coin_type 9000) — bech32. C-chain uses 60.
    9000: "m/44'/9000'/0'/0/%d",
    # Cosmos family — coin_type 118 path, bech32 prefix varies per chain.
    118: "m/44'/118'/0'/0/%d",
    # Testnet alias.
    1: "m/44'/1'/0'/0/%d  (testnet for any chain)",
}


def path_for(coin_type: int) -> str:
    return OVERRIDES.get(coin_type, f"m/44'/{coin_type}'/0'/0/%d")


def render_plain(rows: list[tuple[int, str, str]]) -> str:
    ct_w = max(len(str(r[0])) for r in rows)
    sym_w = min(max(len(r[1]) for r in rows), 12)
    name_w = min(max(len(r[2]) for r in rows), 32)
    lines = []
    for ct, sym, name in rows:
        lines.append(
            f"{ct:>{ct_w}}  {sym[:sym_w]:<{sym_w}}  {name[:name_w]:<{name_w}}  {path_for(ct)}"
        )
    return "\n".join(lines) + "\n"


def render_markdown(rows: list[tuple[int, str, str]]) -> str:
    lines = [
        "| coin_type | ticker | name | derivation path |",
        "|----------:|--------|------|------------------|",
    ]
    for ct, sym, name in rows:
        # escape pipes that might appear in names
        safe_name = name.replace("|", r"\|")
        safe_path = path_for(ct).replace("|", r"\|")
        lines.append(f"| {ct} | {sym} | {safe_name} | `{safe_path}` |")
    return "\n".join(lines) + "\n"


def main() -> int:
    _force_utf8_stdio()
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--source", default=DEFAULT_SOURCE,
                    help="URL or local path to slip-0044.md / constants.json")
    ap.add_argument("--output", default="-",
                    help="file to write ('-' = stdout, default)")
    ap.add_argument("--markdown", action="store_true",
                    help="emit a markdown table instead of fixed-width plain text")
    args = ap.parse_args()

    print(f"[*] fetching {args.source}", file=sys.stderr)
    raw = fetch(args.source)
    rows = parse_any(raw)
    print(f"[*] {len(rows)} coin types parsed, generating derivation paths...", file=sys.stderr)

    text = render_markdown(rows) if args.markdown else render_plain(rows)

    if args.output == "-":
        sys.stdout.write(text)
    else:
        Path(args.output).write_text(text, encoding="utf-8")
        print(f"[+] wrote {args.output} ({len(rows)} chains)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
