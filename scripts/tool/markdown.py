"""Small Markdown helpers for GitHub test summaries."""

import re


def header(text: str, header_level: int) -> str:
    if not 1 <= header_level <= 6:
        raise ValueError("heading level must be between 1 and 6")
    escaped = text.replace("_", r"\_").replace("*", r"\*").replace("`", r"\`")
    return f"{'#' * header_level} {escaped}"


def code_block(text: str, language: str = "") -> str:
    if not language:
        return "\n".join(f"    {line}" for line in text.split("\n"))
    if any(character in language for character in "\r\n`"):
        raise ValueError("code language must not contain newlines or backticks")
    longest_run = max((len(run) for run in re.findall(r"`+", text)), default=0)
    fence = "`" * max(3, longest_run + 1)
    return f"{fence}{language}\n{text}\n{fence}"
