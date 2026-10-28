import unittest

from tool.markdown import code_block, header


class MarkdownTests(unittest.TestCase):
    def test_summary_headings_escape_formatting(self):
        self.assertEqual(header("Graphs", 3), "### Graphs")
        self.assertEqual(
            header("Graph for pkg_name*`graph`", header_level=4),
            r"#### Graph for pkg\_name\*\`graph\`",
        )

    def test_heading_level_must_be_valid(self):
        for level in (0, 7):
            with self.subTest(level=level), self.assertRaises(ValueError):
                header("Graphs", level)

    def test_plain_code_preserves_indentation_and_line_breaks(self):
        self.assertEqual(code_block("# log\n\n  detail\n"), "    # log\n    \n      detail\n    ")
        self.assertEqual(code_block(""), "    ")

    def test_text_graph_uses_fenced_code(self):
        self.assertEqual(code_block("x | **value**", "text"), "```text\nx | **value**\n```")

    def test_content_cannot_close_its_code_fence(self):
        text = "before\n```\n````\nafter"
        self.assertEqual(code_block(text, "text"), f"`````text\n{text}\n`````")

    def test_invalid_language_cannot_break_opening_fence(self):
        for language in ("text\ncode", "text\rcode", "text`code"):
            with self.subTest(language=language), self.assertRaises(ValueError):
                code_block("log", language)


if __name__ == "__main__":
    unittest.main()
