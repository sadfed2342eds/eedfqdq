# scripts/

## update_slip44.py

Pulls the SLIP-0044 registry (canonical coin-type list) and splices it into
the code block between `<!-- SLIP44-BEGIN: ... -->` and `<!-- SLIP44-END -->`
markers in `derivations.md`.

```bash
# default: fetch upstream markdown and write derivations.md in place
python3 scripts/update_slip44.py

# preview changes without writing
python3 scripts/update_slip44.py --dry-run

# CI guard: exit 1 if the file is stale
python3 scripts/update_slip44.py --check

# use a local file or a custom URL (markdown OR bip44-constants JSON is accepted)
python3 scripts/update_slip44.py --source ./slip-0044.md
python3 scripts/update_slip44.py --source https://cdn.jsdelivr.net/gh/satoshilabs/slips@master/slip-0044.md
python3 scripts/update_slip44.py --source https://raw.githubusercontent.com/bitcoinjs/bip44-constants/master/constants.json
```

Only stdlib. Works on Python 3.9+.

A GitHub Actions workflow in `.github/workflows/update-slip44.yml` runs the
script weekly and opens a PR when the registry changed upstream.
