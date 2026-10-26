# Sysutil

Small system utilities, written while learning Go.

Run `sysutil` without arguments to pick a tool from a menu, or add `-i` to any command to be prompted for its options.

| Command                   | Description                                                                     |
| ------------------------- | ------------------------------------------------------------------------------- |
| `ip`                      | Public IP information                                                           |
| `mac [address]`           | MAC vendor lookup (defaults to the active interface, `--list` shows interfaces) |
| `snap`                    | Snapper snapshot manager (TUI, press `?` for keys)                              |
| `ldif to-ldif` / `to-csv` | Convert between CSV and LDIF                                                    |
| `bench`                   | Matrix-vector multiplication benchmark                                          |

## Hints

- `ip`, `mac`, `bench` and `snap list` support `--json`.
- `snap` usually needs root. Scriptable subcommands: `configs`, `list`, `create`, `delete`, `undo`.
- `ldif`: the CSV needs a `dn` column, the default separator is `;`, multiple values are written as `[a][b]`, and `-` means stdin/stdout.
