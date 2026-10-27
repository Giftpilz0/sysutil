# sysutil

[![ci](https://github.com/Giftpilz0/sysutil/actions/workflows/ci.yml/badge.svg)](https://github.com/Giftpilz0/sysutil/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Small system utilities.

## Usage

Run `sysutil` to pick a tool from a menu, or add `-i` to any command to be prompted for its options.

| Command                   | Description                                                   |
| ------------------------- | ------------------------------------------------------------- |
| `ip`                      | Public IP information                                         |
| `mac [address]`           | MAC vendor lookup (active interface by default, `--list`)     |
| `snap`                    | Snapper snapshot manager (TUI, `?` for keys; usually as root) |
| `ldif to-ldif` / `to-csv` | Convert between CSV and LDIF                                  |
| `bench`                   | Matrix-vector multiplication benchmark                        |

- `ip`, `mac`, `bench` and `snap list` support `--json`.
- `ldif`: the CSV needs a `dn` column, separator `;`, multiple values as `[a][b]`, `-` for stdin/stdout.

## Development

Every push to `main` that passes CI is released as `v0.0.N`. Run `just` for all recipes and `just check` before committing. Conventions: [Giftpilz0/shared](https://github.com/Giftpilz0/shared).

## License

[MIT](LICENSE)
