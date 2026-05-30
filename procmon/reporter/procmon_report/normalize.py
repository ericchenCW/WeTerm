"""Derive a stable ``cmdline_key`` from ``(comm, cmdline, cwd, container)``.

The key is what we aggregate by — using ``pid`` alone is unstable across
restarts, and using ``comm`` alone collapses every Java service into a
single bucket. We want something that:

1. Survives restarts (no pid).
2. Distinguishes co-located services (so ``order-svc`` and ``user-svc``
   show as separate rows).
3. Distinguishes container instances when multiple containers run the
   same image.
4. Falls back to ``comm`` when we can't extract anything meaningful.

Priority (highest first):
  1. container present       → ``<container>/<comm>``
  2. java + jar/main-class   → ``java:<name>``
  3. explicit --name X       → ``<comm>:<X>``
  4. cwd present + not default → ``<comm>@<basename-of-cwd>``
  5. fallback                → ``comm``

The rules are intentionally simple regex matches — when something doesn't
match we return ``comm`` rather than guessing wrong.
"""
from __future__ import annotations

import os
import re
from typing import Iterable

# Strip parameters that are routinely sensitive — don't return them as part
# of the key (and don't surface them in the report).
_SENSITIVE_FLAGS = ("--password", "--token", "--secret", "--api-key")


def _extract_java_key(cmdline: str) -> str | None:
    """Find ``-jar X.jar`` or a main class — return ``java:<name>`` or None."""
    tokens = cmdline.split()
    # -jar form: look for the token after -jar
    try:
        i = tokens.index("-jar")
    except ValueError:
        i = -1
    if 0 <= i < len(tokens) - 1:
        jar = tokens[i + 1]
        base = os.path.basename(jar)
        name = re.sub(r"\.jar$", "", base, flags=re.IGNORECASE)
        if name:
            return f"java:{name}"

    # Main class form: scan for a token that looks like a FQCN
    # (contains '.', no '/', no '-X', no leading '-', no '='). We pick
    # the first match — the JVM only takes one anyway.
    for tok in tokens[1:]:  # skip the java binary itself
        if tok.startswith("-") or "=" in tok or "/" in tok:
            continue
        if "." in tok and not tok.endswith(".jar"):
            # Last segment is the class name (matches user expectation in spec).
            cls = tok.rsplit(".", 1)[-1]
            if cls:
                return f"java:{cls}"
    return None


def _extract_name_flag(cmdline: str) -> str | None:
    """Look for ``--name X`` or ``--name=X``. Returns X or None."""
    tokens = cmdline.split()
    for i, tok in enumerate(tokens):
        if tok == "--name" and i + 1 < len(tokens):
            return tokens[i + 1]
        if tok.startswith("--name="):
            return tok.split("=", 1)[1]
    return None


# cwd values that we treat as "no meaningful directory signal" — these
# are typically where a human dropped into a shell, not where a deployed
# service lives. Using basename(`/home/eric`) = "eric" would just pollute
# the key with usernames.
_DEFAULT_ROOTS = {"/", "/root"}


def _is_default_root(cwd: str) -> bool:
    if not cwd:
        return True
    if cwd in _DEFAULT_ROOTS:
        return True
    # Treat any direct child of /home as a user home (no project signal).
    if cwd.startswith("/home/"):
        rest = cwd[len("/home/"):]
        # /home/eric → True; /home/eric/proj-a → False (project subdir keeps signal)
        if "/" not in rest:
            return True
    return False


def normalize_key(
    comm: str,
    cmdline: str,
    cwd: str | None = None,
    container: str | None = None,
) -> str:
    """Return the aggregation key for a process.

    See module docstring for the full priority list. ``cwd`` and ``container``
    are optional — older collectors don't emit them; passing None disables
    the corresponding rules without affecting the rest.
    """
    comm = (comm or "").strip()
    cmdline = (cmdline or "").strip()
    cwd = (cwd or "").strip() if cwd else ""
    container = (container or "").strip() if container else ""

    if container:
        return f"{container}/{comm}"

    if comm == "java" and cmdline:
        key = _extract_java_key(cmdline)
        if key:
            return key

    name = _extract_name_flag(cmdline)
    if name:
        return f"{comm}:{name}"

    if cwd and not _is_default_root(cwd):
        base = os.path.basename(cwd.rstrip("/"))
        if base:
            return f"{comm}@{base}"

    return comm or "?"


def redact_cmdline(cmdline: str) -> str:
    """Replace the *value* of sensitive flags with ``<redacted>``.

    Used when surfacing cmdline in reports for human eyeballs. The raw
    cmdline is still in the JSONL files — this is presentation only.
    """
    tokens = cmdline.split()
    out: list[str] = []
    skip_next = False
    for tok in tokens:
        if skip_next:
            out.append("<redacted>")
            skip_next = False
            continue
        if "=" in tok:
            key, _ = tok.split("=", 1)
            if key in _SENSITIVE_FLAGS:
                out.append(f"{key}=<redacted>")
                continue
        if tok in _SENSITIVE_FLAGS:
            out.append(tok)
            skip_next = True
            continue
        out.append(tok)
    return " ".join(out)


def dump_keys(pairs: Iterable[tuple[str, str]]) -> list[tuple[str, str, str]]:
    """Helper for ``--dump-keys``: returns sorted unique (comm, cmdline, key)."""
    seen: dict[tuple[str, str], str] = {}
    for comm, cmdline in pairs:
        if (comm, cmdline) in seen:
            continue
        seen[(comm, cmdline)] = normalize_key(comm, cmdline)
    return [(c, cl, k) for (c, cl), k in sorted(seen.items())]
