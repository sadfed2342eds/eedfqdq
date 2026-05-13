# BIP44 / HD derivation paths — каталог

Твой текущий путь — `m/44'/6060'/0'/0/%d` (GoChain, SLIP-44 coin type 6060,
EVM-style 20-байтный адрес через keccak256(pub[1:])).

Ниже — максимально полный список известных деривационных шаблонов, который можно
"бить тем же алгоритмом": для всех EVM-цепей (coin_type = 60 или свой) алгоритм
идентичный GoChain'овскому; для не-EVM цепей приведены ещё и особенности
(curve, формат адреса), но **сам BIP32-путь** работает одинаково.

Обозначения:
- `%d` — индекс адреса (обычный, non-hardened).
- `%a` — индекс аккаунта (hardened).
- `'` — hardened child (index + 0x80000000).

Сортировка — по популярности (market cap + частота в живых кошельках, 2025-2026).
Источники: SLIP-0044 registry satoshilabs/slips, BIP-44/49/84/86, Ledger docs,
Phantom/Solflare docs, Trezor firmware coins.json, Trust Wallet core.

---

## 0. Канонические BIP-стандарты (меняется только coin_type)

| BIP   | Префикс           | Назначение                              | Пример для BTC           |
|-------|-------------------|-----------------------------------------|--------------------------|
| 44    | `m/44'/c'/a'/0/i` | Legacy P2PKH / универсальный            | `m/44'/0'/0'/0/0`        |
| 49    | `m/49'/c'/a'/0/i` | P2SH-wrapped SegWit (3... адреса)       | `m/49'/0'/0'/0/0`        |
| 84    | `m/84'/c'/a'/0/i` | Native SegWit, bech32 (bc1q...)         | `m/84'/0'/0'/0/0`        |
| 86    | `m/86'/c'/a'/0/i` | Taproot, bech32m (bc1p...)              | `m/86'/0'/0'/0/0`        |
| 32    | `m/0'/0/i`        | Electrum legacy                         | `m/0'/0/0`               |

---

## 1. Top-50 цепей по популярности

Таблица читается так: `<путь с %d в позиции address_index>` даёт тебе адрес #`%d`
в нулевом аккаунте. Если в кошельке используется другая схема (Ledger Live,
Phantom и т.д.) — это указано в колонке "варианты".

| # | Chain                    | Ticker | SLIP-44 | Основной путь (Metamask/Trust/стандарт) | Альтернативы / примечания |
|---|--------------------------|--------|---------|------------------------------------------|---------------------------|
| 1 | Bitcoin                  | BTC    | 0       | `m/84'/0'/0'/0/%d`                       | `m/44'/0'/0'/0/%d` legacy; `m/49'/0'/0'/0/%d` segwit-compat; `m/86'/0'/0'/0/%d` taproot; `m/0'/0/%d` Electrum |
| 2 | Ethereum + все EVM L1/L2 | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | Ledger Live: `m/44'/60'/%d'/0/0`; Ledger legacy: `m/44'/60'/0'/%d`; testnet: `m/44'/1'/0'/0/%d` |
| 3 | Tether / USDC (on ETH)   | —      | 60      | `m/44'/60'/0'/0/%d`                      | живёт как токен, отдельного пути нет |
| 4 | BNB Smart Chain          | BNB    | 60      | `m/44'/60'/0'/0/%d`                      | как EVM; BNB Beacon (старый): `m/44'/714'/0'/0/%d` |
| 5 | Solana                   | SOL    | 501     | `m/44'/501'/%d'/0'`                      | Phantom/Solflare default; Sollet: `m/44'/501'/%d'`; legacy: `m/501'/0'/0/%d`; ed25519 |
| 6 | XRP (Ripple)             | XRP    | 144     | `m/44'/144'/0'/0/%d`                     | secp256k1 + base58check |
| 7 | USDC (on Solana)         | —      | 501     | `m/44'/501'/%d'/0'`                      | как SOL |
| 8 | Dogecoin                 | DOGE   | 3       | `m/44'/3'/0'/0/%d`                       | - |
| 9 | Cardano                  | ADA    | 1815    | `m/1852'/1815'/%d'/0/0`                  | Shelley payment; staking: `m/1852'/1815'/%d'/2/0`; Byron: `m/44'/1815'/%d'/0/%d`; ed25519 |
| 10| TRON                     | TRX    | 195     | `m/44'/195'/0'/0/%d`                     | secp256k1 + keccak (EVM-подобно) |
| 11| Avalanche C-Chain        | AVAX   | 60      | `m/44'/60'/0'/0/%d`                      | X/P-chain: `m/44'/9000'/0'/0/%d` bech32 |
| 12| TON (The Open Network)   | TON    | 607     | `m/44'/607'/%d'`                         | ed25519; TON wallet v3/v4: семечко -> mnemonic напрямую |
| 13| Polkadot                 | DOT    | 354     | `m/44'/354'/0'/0'/%d'`                   | Sr25519; standard также `//polkadot//%d` |
| 14| Polygon (PoS)            | MATIC  | 60      | `m/44'/60'/0'/0/%d`                      | EVM; historical coin_type 966 не прижился |
| 15| Chainlink                | LINK   | 60      | `m/44'/60'/0'/0/%d`                      | ERC-20 |
| 16| Shiba Inu                | SHIB   | 60      | `m/44'/60'/0'/0/%d`                      | ERC-20 |
| 17| Bitcoin Cash             | BCH    | 145     | `m/44'/145'/0'/0/%d`                     | cashaddr |
| 18| Litecoin                 | LTC    | 2       | `m/84'/2'/0'/0/%d`                       | legacy: `m/44'/2'/0'/0/%d`; segwit-compat: `m/49'/2'/0'/0/%d` |
| 19| Near Protocol            | NEAR   | 397     | `m/44'/397'/0'`                          | ed25519; single-key, index в аккаунте через named account |
| 20| Internet Computer        | ICP    | 223     | `m/44'/223'/0'/0/%d`                     | secp256k1 -> principal |
| 21| Cosmos Hub               | ATOM   | 118     | `m/44'/118'/0'/0/%d`                     | secp256k1, bech32 `cosmos1...` |
| 22| Osmosis                  | OSMO   | 118     | `m/44'/118'/0'/0/%d`                     | bech32 `osmo1...` |
| 23| Celestia                 | TIA    | 118     | `m/44'/118'/0'/0/%d`                     | bech32 `celestia1...` |
| 24| Injective                | INJ    | 60      | `m/44'/60'/0'/0/%d`                      | EVM-compatible Cosmos, bech32 `inj1...` |
| 25| Sei                      | SEI    | 118     | `m/44'/118'/0'/0/%d`                     | bech32 `sei1...` |
| 26| Aptos                    | APT    | 637     | `m/44'/637'/%d'/0'/0'`                   | ed25519 (все сегменты hardened) |
| 27| Sui                      | SUI    | 784     | `m/44'/784'/%d'/0'/0'`                   | ed25519 (как Aptos) |
| 28| Arbitrum                 | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM L2 |
| 29| Optimism                 | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM L2 |
| 30| Base                     | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM L2 (Coinbase) |
| 31| zkSync Era               | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM zkEVM |
| 32| Linea                    | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM zkEVM |
| 33| Scroll                   | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM zkEVM |
| 34| Mantle                   | MNT    | 60      | `m/44'/60'/0'/0/%d`                      | EVM L2 |
| 35| Blast                    | ETH    | 60      | `m/44'/60'/0'/0/%d`                      | EVM L2 |
| 36| Fantom                   | FTM    | 60      | `m/44'/60'/0'/0/%d`                      | EVM; historical 1007 (Nebulas) — не то |
| 37| Cronos                   | CRO    | 60      | `m/44'/60'/0'/0/%d`                      | EVM; Cosmos-сайд: `m/44'/394'/0'/0/%d` |
| 38| Stellar                  | XLM    | 148     | `m/44'/148'/%d'`                         | ed25519, только hardened; seed -> StrKey |
| 39| Monero                   | XMR    | 128     | `m/44'/128'/0'/0/%d`                     | на практике Monero seed это 25-словная mnemonic своего формата |
| 40| Filecoin                 | FIL    | 461     | `m/44'/461'/0'/0/%d`                     | secp256k1 (f1...) |
| 41| Hedera                   | HBAR   | 3030    | `m/44'/3030'/0'/0'/%d'`                  | ed25519 |
| 42| Tezos                    | XTZ    | 1729    | `m/44'/1729'/%d'/0'`                     | ed25519 (tz1...); secp256k1 (tz2...); p256 (tz3...) |
| 43| Algorand                 | ALGO   | 283     | `m/44'/283'/%d'/0'/0'`                   | ed25519 |
| 44| VeChain                  | VET    | 818     | `m/44'/818'/0'/0/%d`                     | EVM-like |
| 45| The Graph                | GRT    | 60      | `m/44'/60'/0'/0/%d`                      | ERC-20 |
| 46| MultiversX (Elrond)      | EGLD   | 508     | `m/44'/508'/0'/0'/%d'`                   | ed25519 |
| 47| Klaytn                   | KLAY   | 8217    | `m/44'/8217'/0'/0/%d`                    | EVM-compatible |
| 48| Flow                     | FLOW   | 539     | `m/44'/539'/0'/0/%d`                     | secp256k1 или p256 (account model) |
| 49| Theta                    | THETA  | 500     | `m/44'/500'/0'/0/%d`                     | EVM-like |
| 50| Kaspa                    | KAS    | 111111  | `m/44'/111111'/0'/0/%d`                  | schnorr/secp256k1; bech32 `kaspa:` |

---

## 2. Популярные EVM-цепи (все используют coin_type 60)

Все работают идентично твоему GoChain коду — меняется только chainID (RPC),
путь остаётся `m/44'/60'/0'/0/%d`:

| Chain           | Ticker | chainID |
|-----------------|--------|---------|
| Ethereum        | ETH    | 1       |
| BNB Smart Chain | BNB    | 56      |
| Polygon PoS     | MATIC  | 137     |
| Avalanche C     | AVAX   | 43114   |
| Fantom          | FTM    | 250     |
| Arbitrum One    | ETH    | 42161   |
| Optimism        | ETH    | 10      |
| Base            | ETH    | 8453    |
| zkSync Era      | ETH    | 324     |
| Linea           | ETH    | 59144   |
| Scroll          | ETH    | 534352  |
| Mantle          | MNT    | 5000    |
| Blast           | ETH    | 81457   |
| Gnosis          | xDAI   | 100     |
| Celo            | CELO   | 42220   |
| Moonbeam        | GLMR   | 1284    |
| Moonriver       | MOVR   | 1285    |
| Harmony         | ONE    | 1666600000 |
| Cronos          | CRO    | 25      |
| Aurora (NEAR)   | ETH    | 1313161554 |
| Fuse            | FUSE   | 122     |
| OKC / OKTC      | OKT    | 66      |
| Metis           | METIS  | 1088    |
| Boba            | BOBA   | 288     |
| KCC             | KCS    | 321     |
| Heco            | HT     | 128 (!) |
| Canto           | CANTO  | 7700    |
| Evmos           | EVMOS  | 9001    |
| Kava EVM        | KAVA   | 2222    |
| Filecoin EVM    | FIL    | 314     |
| opBNB           | BNB    | 204     |
| Manta           | MANTA  | 169     |
| Mode            | ETH    | 34443   |
| **GoChain**     | **GO** | **60 / 6060** | `m/44'/6060'/0'/0/%d` (наш) |
| Ethereum Classic| ETC    | 61      | `m/44'/61'/0'/0/%d` |
| Callisto        | CLO    | 820     | `m/44'/820'/0'/0/%d` |
| ThunderCore     | TT     | 108     | `m/44'/1001'/0'/0/%d` |
| RSK             | RBTC   | 30      | `m/44'/137'/0'/0/%d` (historical!) |
| Ubiq            | UBQ    | 8       | `m/44'/108'/0'/0/%d` |
| Ether-1         | ETHO   | 1313114 | `m/44'/1313114'/0'/0/%d` |
| Ellaism         | ELLA   | 64      | `m/44'/163'/0'/0/%d` |
| Musicoin        | MUSIC  | 7762959 | `m/44'/184'/0'/0/%d` |
| Expanse         | EXP    | 2       | `m/44'/40'/0'/0/%d` |
| POA Network     | POA    | 99      | `m/44'/178'/0'/0/%d` |
| TomoChain       | TOMO   | 88      | `m/44'/889'/0'/0/%d` |
| XDC Network     | XDC    | 50      | `m/44'/550'/0'/0/%d` |
| Wanchain        | WAN    | 888     | `m/44'/5718350'/0'/0/%d` |
| Nervos CKB      | CKB    | —       | `m/44'/309'/0'/0/%d` |

---

## 3. Cosmos-экосистема (все на coin_type 118, отличается только bech32-префикс)

| Chain           | Ticker  | Путь                       | bech32 prefix |
|-----------------|---------|----------------------------|---------------|
| Cosmos Hub      | ATOM    | `m/44'/118'/0'/0/%d`       | `cosmos`      |
| Osmosis         | OSMO    | `m/44'/118'/0'/0/%d`       | `osmo`        |
| Celestia        | TIA     | `m/44'/118'/0'/0/%d`       | `celestia`    |
| Sei             | SEI     | `m/44'/118'/0'/0/%d`       | `sei`         |
| Stride          | STRD    | `m/44'/118'/0'/0/%d`       | `stride`      |
| Juno            | JUNO    | `m/44'/118'/0'/0/%d`       | `juno`        |
| Akash           | AKT     | `m/44'/118'/0'/0/%d`       | `akash`       |
| Secret          | SCRT    | `m/44'/529'/0'/0/%d`       | `secret`      |
| Kava            | KAVA    | `m/44'/459'/0'/0/%d`       | `kava`        |
| Band            | BAND    | `m/44'/494'/0'/0/%d`       | `band`        |
| Terra Classic   | LUNC    | `m/44'/330'/0'/0/%d`       | `terra`       |
| Terra 2.0       | LUNA    | `m/44'/330'/0'/0/%d`       | `terra`       |
| Persistence     | XPRT    | `m/44'/750'/0'/0/%d`       | `persistence` |
| Crypto.org      | CRO     | `m/44'/394'/0'/0/%d`       | `cro`         |
| IRISnet         | IRIS    | `m/44'/118'/0'/0/%d`       | `iaa`         |
| Regen           | REGEN   | `m/44'/118'/0'/0/%d`       | `regen`       |
| Stargaze        | STARS   | `m/44'/118'/0'/0/%d`       | `stars`       |
| Injective       | INJ     | `m/44'/60'/0'/0/%d`        | `inj`         |
| Evmos           | EVMOS   | `m/44'/60'/0'/0/%d`        | `evmos`       |
| Desmos          | DSM     | `m/44'/852'/0'/0/%d`       | `desmos`      |
| BitCanna        | BCNA    | `m/44'/118'/0'/0/%d`       | `bcna`        |
| Sifchain        | ROWAN   | `m/44'/118'/0'/0/%d`       | `sif`         |

---

## 4. Bitcoin-форки и LTC/DOGE-семейство

| Coin                | Ticker | SLIP-44 | Пример пути          |
|---------------------|--------|---------|-----------------------|
| Bitcoin             | BTC    | 0       | `m/84'/0'/0'/0/%d`    |
| Bitcoin Cash        | BCH    | 145     | `m/44'/145'/0'/0/%d`  |
| Bitcoin SV          | BSV    | 236     | `m/44'/236'/0'/0/%d`  |
| Bitcoin Gold        | BTG    | 156     | `m/44'/156'/0'/0/%d`  |
| Bitcoin Diamond     | BCD    | 999     | `m/44'/999'/0'/0/%d`  |
| Bitcoin Private     | BTCP   | 183     | `m/44'/183'/0'/0/%d`  |
| Bitcoin Atom        | BCA    | 185     | `m/44'/185'/0'/0/%d`  |
| BitCore             | BTX    | 160     | `m/44'/160'/0'/0/%d`  |
| Litecoin            | LTC    | 2       | `m/84'/2'/0'/0/%d`    |
| Litecoin Cash       | LCC    | 192     | `m/44'/192'/0'/0/%d`  |
| Dogecoin            | DOGE   | 3       | `m/44'/3'/0'/0/%d`    |
| Dash                | DASH   | 5       | `m/44'/5'/0'/0/%d`    |
| Digibyte            | DGB    | 20      | `m/44'/20'/0'/0/%d`   |
| Monacoin            | MONA   | 22      | `m/44'/22'/0'/0/%d`   |
| Vertcoin            | VTC    | 28      | `m/44'/28'/0'/0/%d`   |
| Groestlcoin         | GRS    | 17      | `m/84'/17'/0'/0/%d`   |
| Peercoin            | PPC    | 6       | `m/44'/6'/0'/0/%d`    |
| Namecoin            | NMC    | 7       | `m/44'/7'/0'/0/%d`    |
| Reddcoin            | RDD    | 4       | `m/44'/4'/0'/0/%d`    |
| Blackcoin           | BLK    | 10      | `m/44'/10'/0'/0/%d`   |
| Feathercoin         | FTC    | 8       | `m/44'/8'/0'/0/%d`    |
| Viacoin             | VIA    | 14      | `m/44'/14'/0'/0/%d`   |
| Syscoin             | SYS    | 57      | `m/44'/57'/0'/0/%d`   |
| Verge               | XVG    | 77      | `m/44'/77'/0'/0/%d`   |
| Zcash               | ZEC    | 133     | `m/44'/133'/0'/0/%d`  |
| Zclassic            | ZCL    | 147     | `m/44'/147'/0'/0/%d`  |
| Horizen             | ZEN    | 121     | `m/44'/121'/0'/0/%d`  |
| Komodo              | KMD    | 141     | `m/44'/141'/0'/0/%d`  |
| PIVX                | PIVX   | 119     | `m/44'/119'/0'/0/%d`  |
| Firo (Zcoin)        | FIRO   | 136     | `m/44'/136'/0'/0/%d`  |
| Ravencoin           | RVN    | 175     | `m/44'/175'/0'/0/%d`  |
| Particl             | PART   | 44      | `m/44'/44'/0'/0/%d`   |
| Decred              | DCR    | 42      | `m/44'/42'/0'/0/%d`   |
| Nano                | XNO    | 165     | `m/44'/165'/0'/0/%d`  |
| Banano              | BAN    | 198     | `m/44'/198'/0'/0/%d`  |
| Beam                | BEAM   | 2019    | `m/44'/2019'/0'/0/%d` |
| Grin                | GRIN   | 592     | `m/44'/592'/0'/0/%d`  |
| Handshake           | HNS    | 5353    | `m/44'/5353'/0'/0/%d` |
| LBRY                | LBC    | 140     | `m/44'/140'/0'/0/%d`  |
| MimbleWimbleCoin    | MWC    | 592     | `m/44'/592'/0'/0/%d`  |

---

## 5. ED25519 / специальные кривые и нестандартные пути

Эти цепи **не используют secp256k1**, но BIP32-дерево по mnemonic всё равно
строится. Путь часто выглядит непривычно (все сегменты hardened, или
единственный сегмент):

| Chain       | Ticker | Curve      | Путь                              |
|-------------|--------|------------|-----------------------------------|
| Solana      | SOL    | ed25519    | `m/44'/501'/%d'/0'`               |
| Stellar     | XLM    | ed25519    | `m/44'/148'/%d'`                  |
| Near        | NEAR   | ed25519    | `m/44'/397'/0'`                   |
| Aptos       | APT    | ed25519    | `m/44'/637'/%d'/0'/0'`            |
| Sui         | SUI    | ed25519    | `m/44'/784'/%d'/0'/0'`            |
| Algorand    | ALGO   | ed25519    | `m/44'/283'/%d'/0'/0'`            |
| Tezos       | XTZ    | ed25519    | `m/44'/1729'/%d'/0'`              |
| TON         | TON    | ed25519    | `m/44'/607'/%d'`                  |
| MultiversX  | EGLD   | ed25519    | `m/44'/508'/0'/0'/%d'`            |
| Hedera      | HBAR   | ed25519    | `m/44'/3030'/0'/0'/%d'`           |
| Polkadot    | DOT    | sr25519    | `m/44'/354'/0'/0'/%d'` или `//polkadot//%d` |
| Kusama      | KSM    | sr25519    | `m/44'/434'/0'/0'/%d'` или `//kusama//%d`   |
| Cardano     | ADA    | ed25519    | `m/1852'/1815'/%d'/0/0` (Shelley payment); `m/1852'/1815'/%d'/2/0` (stake) |
| IOTA        | MIOTA  | ed25519    | `m/44'/4218'/%d'/0'/0'`           |
| Shimmer     | SMR    | ed25519    | `m/44'/4219'/%d'/0'/0'`           |
| MINA        | MINA   | pallas     | `m/44'/12586'/%d'/0/0`            |
| Zilliqa     | ZIL    | schnorr    | `m/44'/313'/0'/0/%d`              |
| NEO         | NEO    | NIST p256  | `m/44'/888'/0'/0/%d`              |
| Ontology    | ONT    | NIST p256  | `m/44'/1024'/0'/0/%d`             |
| Waves       | WAVES  | curve25519 | `m/44'/5741564'/0'/0/%d` (исторически) |
| Radix       | XRD    | secp256k1  | `m/44'/1022'/0'/0/%d` или `m/44'/1022'/%d'/525'/1460'/0'` |

---

## 6. Ledger Live / Trezor-специфические варианты

Ledger Live *по умолчанию* для некоторых цепей отступает от стандарта —
увеличивает `account'` вместо `address_index`:

| Chain    | Ledger Live default          | Стандарт (BIP44)            |
|----------|------------------------------|------------------------------|
| Ethereum | `m/44'/60'/%d'/0/0`          | `m/44'/60'/0'/0/%d`          |
| ETC      | `m/44'/60'/%d'/0/0`          | `m/44'/61'/0'/0/%d`          |
| EOS      | `m/44'/194'/%d'/0/0`         | `m/44'/194'/0'/0/%d`         |
| Ripple   | `m/44'/144'/%d'/0/0`         | `m/44'/144'/0'/0/%d`         |
| Solana   | `m/44'/501'/%d'`             | `m/44'/501'/%d'/0'` (Phantom)|

Legacy Ledger ETH пути (до Ledger Live): `m/44'/60'/0'/%d`, `m/44'/60'/0'/0/%d`,
`m/44'/60'/%d'`.

---

## 7. Длинный "хвост" SLIP-44 coin types (альткоины, референс)

Формат записи: `coin_type | ticker | name`, путь везде `m/44'/<coin_type>'/0'/0/%d`
если не указано иное.

```
 0    BTC     Bitcoin
 1    -       Testnet (все цепи)
 2    LTC     Litecoin
 3    DOGE    Dogecoin
 4    RDD     Reddcoin
 5    DASH    Dash
 6    PPC     Peercoin
 7    NMC     Namecoin
 8    FTC     Feathercoin
 9    XCP     Counterparty
 10   BLK     Blackcoin
 11   NSR     NuShares
 12   NBT     NuBits
 13   MZC     Mazacoin
 14   VIA     Viacoin
 15   XCH     (reserved)
 16   RBY     Rubycoin
 17   GRS     Groestlcoin
 18   DGC     Digitalcoin
 19   CCN     Cannacoin
 20   DGB     DigiByte
 22   MONA    Monacoin
 23   CLAM    Clams
 24   XPM     Primecoin
 25   NEOS    Neoscoin
 26   JBS     Jumbucks
 28   VTC     Vertcoin
 29   NXT     NXT
 30   BURST   Burst
 31   MUE     MonetaryUnit
 33   VASH    Virtual Cash
 35   SDC     ShadowCash
 36   PKB     ParkByte
 37   PND     Pandacoin
 38   START   StartCoin
 40   EXP     Expanse
 41   EMC2    Einsteinium
 42   DCR     Decred
 43   XEM     NEM
 44   PART    Particl
 50   NVC     Novacoin
 52   BTCD    BitcoinDark
 57   SYS     Syscoin
 60   ETH     Ethereum
 61   ETC     Ethereum Classic
 74   ICX     ICON
 77   XVG     Verge
 88   NLG     Gulden
 101   GAME   GameCredits
 105   STRAT  Stratis
 108   UBQ    Ubiq
 111   LINX   Linx  (или ARK=111 — см.)
 118   ATOM   Cosmos
 119   PIVX   PIVX
 121   ZEN    Horizen
 128   XMR    Monero
 133   ZEC    Zcash
 134   LSK    Lisk
 135   STEEM  Steem
 136   FIRO   Firo (XZC)
 137   RSK    RSK / RBTC
 140   LBC    LBRY
 141   KMD    Komodo
 144   XRP    Ripple
 145   BCH    Bitcoin Cash
 147   ZCL    Zclassic
 148   XLM    Stellar
 153   BTM    Bytom
 156   BTG    Bitcoin Gold
 160   BTX    BitCore
 165   XNO    Nano
 175   RVN    Ravencoin
 178   POA    POA Network
 183   BTCP   Bitcoin Private
 185   BCA    Bitcoin Atom
 192   LCC    Litecoin Cash
 194   EOS    EOS
 195   TRX    TRON
 198   BAN    Banano
 199   OMNI   Omni
 207   NEET   Neetcoin
 222   ICP    Internet Computer
 223   -      ICP (альт)
 235   BSV    Bitcoin SV
 236   DXN    Dexon
 237   QRL    Quantum Resistant Ledger
 238   PCX    ChainX
 239   LOKI   Oxen (ex-Loki)
 242   NIM    Nimiq
 283   ALGO   Algorand
 304   IOTX   IoTeX
 309   CKB    Nervos CKB
 313   ZIL    Zilliqa
 330   LUNA   Terra
 354   DOT    Polkadot
 367   THETA  Theta (также 500)
 394   CRO    Crypto.org
 397   NEAR   NEAR Protocol
 425   SIN    SINOVATE
 434   KSM    Kusama
 459   KAVA   Kava
 461   FIL    Filecoin
 474   KAI    KardiaChain
 489   HNS    Handshake (также 5353)
 494   BAND   Band Protocol
 500   THETA  Theta Network
 501   SOL    Solana
 508   EGLD   MultiversX (Elrond)
 510   XDC    XinFin
 529   SCRT   Secret Network
 539   FLOW   Flow
 550   AE     Aeternity
 589   HNT    Helium (также 594)
 592   GRIN   Grin
 594   HNT    Helium
 607   TON    The Open Network / TFUEL
 637   APT    Aptos
 700   xDAI   Gnosis Chain (ранее xDAI)
 703   VLX    Velas
 714   BNB    BNB Beacon Chain
 731   STX    Stacks
 750   XPRT   Persistence
 784   SUI    Sui
 818   VET    VeChain
 820   CLO    Callisto
 852   DSM    Desmos
 888   NEO    NEO
 889   TOMO   TomoChain
 904   HPB    High Performance Blockchain
 931   RUNE   THORChain
 996   OKT    OKC / OKExChain
 999   BCD    Bitcoin Diamond
 1007  FTM    Fantom (исторически; сейчас 60)
 1008  VITE   Vite
 1022  XRD    Radix
 1023  ONE    Harmony
 1024  ONT    Ontology
 1729  XTZ    Tezos
 1815  ADA    Cardano
 1899  HSN    Hyper Speed Network
 1984  UBT    UBT
 2017  KTO    Kto
 2020  XRD    Radix (старый)
 2301  QTUM   Qtum
 2303  GXC    GXChain
 2304  ELA    Elastos
 2718  NAS    Nebulas
 3030  HBAR   Hedera
 4218  IOTA   IOTA
 4219  SMR    Shimmer
 5353  HNS    Handshake
 5741564 WAVES исторический
 5718350 WAN  Wanchain
 6060  GO     GoChain   <--- ТВОЙ ПУТЬ
 7777777 FIO  FIO
 8217  KLAY   Klaytn
 9000  AVAX   Avalanche X-/P-chain
 12586 MINA   Mina
 111111 KAS   Kaspa
 1313114 ETHO Ether-1
 1313161554 AURORA Aurora (NEAR EVM)
```

---

## 8. Go-константы — быстро подключить к твоему парсеру

Если нужно пачкой бить тот же код по нескольким цепям — достаточно менять
coin_type. Для EVM-цепей **адрес получается тем же keccak-методом**, что
в твоём `privToAddress` (secp256k1 uncompressed pub[1:] -> keccak256 -> 20
последних байт). Для ED25519-цепей нужен другой код (другая кривая и
другой hash-to-address), файл ниже это не решает — только пути.

```go
// derivations.go
package derivations

// Path: m/44'/CoinType'/0'/0/%d
type Chain struct {
	Name     string
	Ticker   string
	CoinType uint32
	EVM      bool // если true — адрес = keccak(secp256k1_pub)[12:]
}

var Top = []Chain{
	{"Bitcoin",           "BTC",   0,    false},
	{"Ethereum",          "ETH",   60,   true},
	{"GoChain",           "GO",    6060, true},  // <-- ты здесь
	{"Ethereum Classic",  "ETC",   61,   true},
	{"Callisto",          "CLO",   820,  true},
	{"TomoChain",         "TOMO",  889,  true},
	{"XDC",               "XDC",   550,  true},
	{"Expanse",           "EXP",   40,   true},
	{"Ubiq",              "UBQ",   108,  true},
	{"POA",               "POA",   178,  true},
	{"Musicoin",          "MUSIC", 184,  true},
	{"Ellaism",           "ELLA",  163,  true},
	{"Ether-1",           "ETHO",  1313114, true},
	{"Wanchain",          "WAN",   5718350, true},
	{"RSK",               "RBTC",  137,  true},
	// — не-EVM (только путь, адрес считается иначе):
	{"Ripple",            "XRP",   144,  false},
	{"Bitcoin Cash",      "BCH",   145,  false},
	{"Litecoin",          "LTC",   2,    false},
	{"Dogecoin",          "DOGE",  3,    false},
	{"Dash",              "DASH",  5,    false},
	{"Zcash",             "ZEC",   133,  false},
	{"Monero",            "XMR",   128,  false},
	{"Cosmos",            "ATOM",  118,  false},
	{"Terra",             "LUNA",  330,  false},
	{"TRON",              "TRX",   195,  false}, // secp256k1 + keccak как EVM, но base58
	{"Filecoin",          "FIL",   461,  false},
	{"Kaspa",             "KAS",   111111, false},
}
```

---

## 9. Одной фразой

- Если хочешь **тот же код что для GoChain (EVM-стиль) применить на другую
  сеть** — меняй только `6060` на coin_type из таблицы § 2 (60 для всех
  Ethereum-совместимых).
- Если нужно **адреса в других экосистемах** — путь из таблицы § 1 / § 5, но
  `privToAddress` надо переписать под соответствующую кривую/формат
  (bech32 / base58check / ed25519-public / cashaddr и т.д.).
- Самые ходовые пути на практике, по убыванию:
  1. `m/44'/60'/0'/0/%d` (ETH и ~50 EVM-цепей)
  2. `m/84'/0'/0'/0/%d` (BTC native SegWit)
  3. `m/44'/501'/%d'/0'` (SOL)
  4. `m/44'/0'/0'/0/%d` (BTC legacy)
  5. `m/44'/144'/0'/0/%d` (XRP)
  6. `m/44'/195'/0'/0/%d` (TRX)
  7. `m/1852'/1815'/%d'/0/0` (ADA)
  8. `m/44'/118'/0'/0/%d` (вся Cosmos-экосистема)
  9. `m/44'/397'/0'` (NEAR)
  10. `m/44'/461'/0'/0/%d` (FIL)
