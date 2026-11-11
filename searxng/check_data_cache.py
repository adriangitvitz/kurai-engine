#!/usr/bin/env python3

import sqlite3
import sys

DB_PATH = "/tmp/sxng_cache_DATA_CACHE.db"

CURRENCY_CONTEXTS = ("data_currencies_names", "data_currencies_iso4217")
TRACKER_CONTEXT = "data_tracker_patterns"


def row_count(context: str) -> int:
    try:
        con = sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True)
    except sqlite3.OperationalError:
        return 0
    try:
        return con.execute(f'SELECT count(*) FROM "{context}"').fetchone()[0]
    except sqlite3.OperationalError:
        return 0
    finally:
        con.close()


def main() -> int:
    currencies_empty = any(row_count(c) == 0 for c in CURRENCY_CONTEXTS)
    tracker_empty = row_count(TRACKER_CONTEXT) == 0

    if not currencies_empty and not tracker_empty:
        return 0

    if currencies_empty:
        print("data cache: currency contexts empty, reloading", file=sys.stderr)
        from searx.data import CURRENCIES

        try:
            CURRENCIES.load()
        except Exception as exc:  # noqa: BLE001 - report, don't crash the check
            print(f"data cache: currency reload failed: {exc}", file=sys.stderr)

    if tracker_empty:
        print("data cache: tracker patterns empty, reloading", file=sys.stderr)
        from searx.data import TRACKER_PATTERNS

        try:
            TRACKER_PATTERNS.load()
        except Exception as exc:  # noqa: BLE001
            print(f"data cache: tracker reload failed: {exc}", file=sys.stderr)

    if any(row_count(c) == 0 for c in CURRENCY_CONTEXTS):
        print("data cache: currency contexts still empty after reload", file=sys.stderr)
        return 1
    if row_count(TRACKER_CONTEXT) == 0:
        print(
            "data cache: tracker patterns still empty after reload "
            "(third-party source; will retry next interval)",
            file=sys.stderr,
        )
    return 0


if __name__ == "__main__":
    sys.exit(main())
