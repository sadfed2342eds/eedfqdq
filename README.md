# Fast GoChain (BIP44 coin type 6060) mnemonic parser

Многопоточная деривация адресов по пути `m/44'/6060'/0'/0/%d` из списка BIP39 мнемоник.
Coin type 6060 = GoChain; адреса EVM-совместимые (secp256k1 + keccak256).

## Вход / Выход

- `seed.txt` - одна мнемоника на строку.
- `result.txt` - `<address>|<privkey_hex>|<mnemonic>` (одна строка на адрес).
- `adress.txt` - только адреса.

## Сборка (на твоей машине с интернетом)

```bash
go mod tidy
go build -ldflags="-s -w" -o parser.exe .
```

Нужен Go >= 1.22. На Windows Server 2019 / Ryzen 9 5950X собирай так же
(`go build -o parser.exe .`).

## Запуск

```bash
# минимум: один адрес (индекс 0) на каждую мнемонику
./parser.exe -in seed.txt -out result.txt -addr adress.txt

# 10 адресов (индексы 0..9) на каждую мнемонику, 32 воркера
./parser.exe -in seed.txt -n 10 -w 32
```

## Флаги

| Флаг  | Default        | Смысл                                   |
|-------|----------------|-----------------------------------------|
| `-in` | `seed.txt`     | входной файл                            |
| `-out`| `result.txt`   | результат (addr\|priv\|mnemonic)        |
| `-addr`| `adress.txt`  | только адреса                           |
| `-n`  | `1`            | сколько индексов на мнемонику (0..n-1)  |
| `-w`  | `NumCPU()`     | число воркеров (на 5950X = 32)          |

## Почему быстро

- На мнемонику полный путь `m/44'/6060'/0'/0` (4 hardened + 1 non-hardened
  деривации) считается **один раз**. Для каждого индекса дальше - всего одна
  HMAC-SHA512 + одно сложение по модулю n + одно умножение на базовую точку
  secp256k1 + один keccak.
- HMAC-объекты и keccak-хешер переиспользуются через `Reset()` - нет аллокаций
  в горячем цикле.
- secp256k1 - `decred/dcrd/dcrec/secp256k1/v4` (чистый Go, без cgo, но с
  оптимизированной скалярной арифметикой).
- Writer сидит в отдельной горутине с `bufio.Writer` 1 MiB - диск не
  блокирует счётные воркеры.
- Прогресс печатается в stderr каждые 5 секунд.

## Ожидаемая производительность на 5950X

- При `-n 1`: около 150-300k мнемоник/с (2 scalar_mul на мнемонику).
- При `-n >= 100`: ~1.5-2M адресов/с (амортизируется фиксированная стоимость
  hardened деривации).

Реальные цифры сильно зависят от длины мнемоник (12 vs 24 слова, это стоимость
PBKDF2 внутри `bip39.NewSeed`) и скорости диска для output.



---

## Checker (GoChain native balance)

Нативный чеккер (только stdlib). Pipeline из 4 этапов:

```
[input] -> activity-workers -> aggregator -> balance-workers -> writer
```

1. **Activity** (фаза 1): батч `eth_getTransactionCount` + `eth_getCode`
   (2*N вызовов в одном HTTP round-trip). Активный = `nonce > 0` ИЛИ контракт.
2. **Aggregator**: собирает поток активных адресов из всех phase-1 воркеров
   в крупные батчи (по `-bb`, default 500) с принудительным flush по
   таймеру (`-flush` ms). Phase 2 всегда работает большими батчами.
3. **Balance** (фаза 2): батч `eth_getBalance` для активных.
4. **Writer**: один bufio.Writer 1 MiB в отдельной горутине.

В `result.txt` попадают все активные адреса (включая с 0 балансом).

### Сборка

```bash
go build -ldflags="-s -w" -o checker.exe ./cmd/checker
```

### Запуск

```bash
# дефолты: in=adress.txt, out=result.txt, публичные rpc.gochain.io/org
checker.exe

# своя нода, агрессивные параметры
checker.exe -rpc https://my-node:8545 -wa 128 -wb 64 -ab 500 -bb 1000
```

### Формат `result.txt`

```
<address> <wei> <GO> nonce=<n> contract=<bool>
```

Пример:

```
0xd59be811e3c974a05fcd8136845e745b208c134d 1234000000000000000 1.234 nonce=7 contract=false
0x0000000000000000000000000000000000001337 0 0 nonce=0 contract=true
```

### Флаги

| Флаг        | Default                                            | Смысл                                     |
|-------------|----------------------------------------------------|-------------------------------------------|
| `-in`       | `adress.txt`                                       | файл со списком адресов                   |
| `-out`      | `result.txt`                                       | вывод активных адресов                    |
| `-rpc`      | `https://rpc.gochain.io,https://rpc.gochain.org`   | RPC endpoints через запятую (round-robin) |
| `-wa`       | `NumCPU()*4`                                       | activity-phase воркеры                    |
| `-wb`       | `NumCPU()*2`                                       | balance-phase воркеры                     |
| `-ab`       | `200`                                              | адресов в activity-батче                  |
| `-bb`       | `500`                                              | адресов в balance-батче                   |
| `-flush`    | `500`                                              | мс до flush неполного balance-батча       |
| `-timeout`  | `30s`                                              | HTTP timeout                              |
| `-retries`  | `3`                                                | ретраи на сетевую/HTTP ошибку             |

### Замечания

- Публичные RPC имеют рейт-лимит. Для больших списков бери свою ноду.
- Несколько URL через запятую - воркеры размазываются round-robin.
- Тюнинг: на публичном RPC начни с `-wa 8 -wb 4 -ab 50 -bb 100`,
  на своей ноде можно `-wa 256 -wb 128 -ab 500 -bb 1000`.
