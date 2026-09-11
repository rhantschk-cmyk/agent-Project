#!/usr/bin/env python3
"""
VaultAgent — CLI tool (server-side) for the Standard and Pro editions.

Compatibility shim — the actual implementation lives in cli.agent_cli.
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from cli.agent_cli import main  # noqa: E402

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\n\nAborted.")
        sys.exit(130)