from __future__ import annotations

import base64
import hashlib
import json
import subprocess
import shutil
import tempfile
import unittest
import zipfile
from pathlib import Path
from xml.etree import ElementTree


REPO_ROOT = Path(__file__).resolve().parents[1]
PYTHON = Path(
    "/Users/sak/.cache/codex-runtimes/codex-primary-runtime/"
    "dependencies/python/bin/python3"
)
MARKDOWN_EXPORTER = REPO_ROOT / "tools" / "export_feishu_guides.py"
WORD_EXPORTER = REPO_ROOT / "tools" / "export_word_guides.py"
CONTENT_DIR = REPO_ROOT / "frontend" / "src" / "content" / "guides-v2"
OUTPUT_PARENT = "02-AI客户端使用指南"
STEMS = (
    "00-教程中心",
    "01-快速开始",
    "02-Codex",
    "03-Claude-Code",
    "04-OpenCode",
    "05-OpenClaw",
    "06-Chatbox-移动端",
    "07-Cherry-Studio-生图",
    "08-公共排错中心",
)
MD_OUTPUTS = {f"{OUTPUT_PARENT}/{stem}.md" for stem in STEMS}
DOCX_OUTPUTS = {f"{OUTPUT_PARENT}/{stem}.docx" for stem in STEMS}
WORD_NS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
REL_NS = "http://schemas.openxmlformats.org/package/2006/relationships"


def run_export(
    script: Path,
    output_dir: Path,
    *extra_args: str,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [
            str(PYTHON),
            str(script),
            "--edition",
            "v2",
            "--output-dir",
            str(output_dir),
            *extra_args,
        ],
        cwd=REPO_ROOT,
        capture_output=True,
        text=True,
        check=False,
    )


def document_text(path: Path) -> str:
    with zipfile.ZipFile(path) as archive:
        root = ElementTree.fromstring(archive.read("word/document.xml"))
    return "\n".join(
        "".join(node.text or "" for node in paragraph.iter(f"{{{WORD_NS}}}t"))
        for paragraph in root.iter(f"{{{WORD_NS}}}p")
    )


class V2GuideExportTests(unittest.TestCase):
    def test_manifest_hashes_match_all_nine_sources(self) -> None:
        manifest = json.loads(
            (CONTENT_DIR / "manifest.generated.json").read_text("utf-8")
        )
        entries = manifest["entries"]
        self.assertEqual(len(entries), 9)
        for entry in entries:
            source = CONTENT_DIR / entry["source"]
            source_text = source.read_text("utf-8")
            body = source_text.split("\n---\n", 1)[1]
            with self.subTest(source=source.name):
                self.assertEqual(
                    hashlib.sha256(body.encode("utf-8")).hexdigest(),
                    entry["contentHash"],
                )

    def test_exporter_rejects_stale_source_hash_and_source_symlink(self) -> None:
        from tools import export_guides_v2 as exporter

        with tempfile.TemporaryDirectory() as temporary_directory:
            content = Path(temporary_directory) / "guides-v2"
            shutil.copytree(CONTENT_DIR, content)
            stale_source = content / "codex.md"
            stale_source.write_text(stale_source.read_text("utf-8") + "\n", "utf-8")
            with self.assertRaisesRegex(ValueError, "pnpm --dir frontend guides:v2:manifest"):
                exporter.load_verified_bodies(content)

            stale_source.unlink()
            stale_source.symlink_to(CONTENT_DIR / "codex.md")
            with self.assertRaisesRegex(ValueError, "拒绝符号链接"):
                exporter.load_verified_bodies(content)

    def test_v2_exports_exact_nine_frontmatter_free_markdown_guides(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = run_export(MARKDOWN_EXPORTER, output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)
            outputs = {
                path.relative_to(output_dir).as_posix()
                for path in output_dir.rglob("*.md")
            }
            self.assertEqual(outputs, MD_OUTPUTS)
            for output in outputs:
                content = (output_dir / output).read_text("utf-8")
                with self.subTest(output=output):
                    self.assertFalse(content.startswith("---"))
                    self.assertNotIn("\nslug:", content)
                    self.assertNotIn("\nupdatedAt:", content)
                    self.assertNotIn("{#", content)

    def test_v2_markdown_uses_png_data_uris_instead_of_webp(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = run_export(MARKDOWN_EXPORTER, output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)
            codex = (output_dir / OUTPUT_PARENT / "02-Codex.md").read_text("utf-8")
            self.assertNotIn(".webp", codex)
            encoded = codex.split("data:image/png;base64,", 1)[1].split(")", 1)[0]
            self.assertTrue(base64.b64decode(encoded, validate=True).startswith(b"\x89PNG"))

    def test_v2_word_exports_embed_images_and_preserve_platform_branches(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = run_export(WORD_EXPORTER, output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)
            outputs = {
                path.relative_to(output_dir).as_posix()
                for path in output_dir.rglob("*.docx")
            }
            self.assertEqual(outputs, DOCX_OUTPUTS)
            total_media = 0
            for output in outputs:
                docx_path = output_dir / output
                with zipfile.ZipFile(docx_path) as archive:
                    self.assertIsNone(archive.testzip())
                    media = [
                        name
                        for name in archive.namelist()
                        if name.startswith("word/media/")
                    ]
                    total_media += len(media)
                    relationships = ElementTree.fromstring(
                        archive.read("word/_rels/document.xml.rels")
                    )
                    external_images = [
                        relation
                        for relation in relationships.iter(
                            f"{{{REL_NS}}}Relationship"
                        )
                        if relation.attrib.get("TargetMode") == "External"
                        and relation.attrib.get("Type", "").endswith("/image")
                    ]
                    self.assertFalse(external_images)
                    for media_path in media:
                        self.assertTrue(
                            archive.read(media_path).startswith(b"\x89PNG\r\n\x1a\n")
                        )
                    xml = "\n".join(
                        archive.read(name).decode("utf-8")
                        for name in archive.namelist()
                        if name.endswith(".xml")
                    )
                    self.assertIn('w:eastAsia="Hiragino Sans GB"', xml)
                    self.assertNotIn('w:eastAsia="PingFang SC"', xml)
                text = document_text(docx_path)
                self.assertNotIn("slug:", text)
                self.assertNotIn("updatedAt:", text)
                self.assertNotIn("SAK AI", text)
                self.assertIn("AI 客户端使用指南", text)
                self.assertNotIn("{#", text)
            self.assertGreater(total_media, 0)

            codex = document_text(output_dir / OUTPUT_PARENT / "02-Codex.docx")
            self.assertTrue(all(label in codex for label in ("Windows", "macOS", "Linux")))
            openclaw = document_text(output_dir / OUTPUT_PARENT / "05-OpenClaw.docx")
            self.assertIn("腾讯云", openclaw)
            self.assertIn("本地", openclaw)
            chatbox = document_text(
                output_dir / OUTPUT_PARENT / "06-Chatbox-移动端.docx"
            )
            self.assertIn("iOS", chatbox)
            self.assertIn("Android", chatbox)

    def test_v2_check_detects_missing_extra_and_stale_without_writing(self) -> None:
        for script, extension in (
            (MARKDOWN_EXPORTER, ".md"),
            (WORD_EXPORTER, ".docx"),
        ):
            with self.subTest(script=script.name), tempfile.TemporaryDirectory() as temporary_directory:
                output_dir = Path(temporary_directory)
                generated = run_export(script, output_dir)
                self.assertEqual(generated.returncode, 0, generated.stderr)
                expected = sorted(output_dir.rglob(f"*{extension}"))

                current = run_export(script, output_dir, "--check")
                self.assertEqual(current.returncode, 0, current.stderr)

                missing = expected[0]
                missing_bytes = missing.read_bytes()
                missing.unlink()
                missing_result = run_export(script, output_dir, "--check")
                self.assertNotEqual(missing_result.returncode, 0)
                self.assertIn(missing.name, missing_result.stderr)
                self.assertFalse(missing.exists())

                missing.write_bytes(missing_bytes)
                extra = missing.parent / f"extra{extension}"
                extra.write_bytes(missing_bytes)
                extra_result = run_export(script, output_dir, "--check")
                self.assertNotEqual(extra_result.returncode, 0)
                self.assertIn(extra.name, extra_result.stderr)
                self.assertTrue(extra.exists())

                extra.unlink()
                missing.write_bytes(missing_bytes + b"stale")
                stale_result = run_export(script, output_dir, "--check")
                self.assertNotEqual(stale_result.returncode, 0)
                self.assertIn(missing.name, stale_result.stderr)
                self.assertEqual(missing.read_bytes(), missing_bytes + b"stale")

    def test_video_is_degraded_to_poster_explanation_and_link(self) -> None:
        from tools import export_guides_v2 as exporter

        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            poster = root / "poster.png"
            poster.write_bytes(b"\x89PNG\r\n\x1a\n" + b"\0" * 25)
            rendered = exporter.degrade_video(
                '<video poster="/poster.webp" src="https://example.com/demo.mp4"></video>',
                poster_data_uri="data:image/png;base64,AAAA",
                alt="配置演示视频",
                caption="演示完整配置流程。",
                link="https://example.com/demo.mp4",
            )
            self.assertIn("data:image/png;base64,AAAA", rendered)
            self.assertIn("视频说明：演示完整配置流程。", rendered)
            self.assertIn("视频链接：https://example.com/demo.mp4", rendered)
            self.assertNotIn("<video", rendered)

    def test_v2_transaction_rejects_escape_and_symlink_without_partial_output(self) -> None:
        from tools import export_guides_v2 as exporter

        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            output = root / "output"
            output.mkdir()
            sentinel = output / "sentinel.txt"
            sentinel.write_text("keep", "utf-8")
            with self.assertRaises(ValueError):
                exporter.atomic_export_tree(
                    output,
                    (("safe/one.md", b"one"), ("../escaped.md", b"bad")),
                )
            self.assertEqual(sentinel.read_text("utf-8"), "keep")
            self.assertFalse((root / "escaped.md").exists())
            self.assertFalse((output / "safe" / "one.md").exists())

            outside = root / "outside"
            outside.mkdir()
            linked_output = root / "linked-output"
            linked_output.symlink_to(outside, target_is_directory=True)
            with self.assertRaises(ValueError):
                exporter.atomic_export_tree(
                    linked_output,
                    ((f"{OUTPUT_PARENT}/one.md", b"one"),),
                )
            self.assertEqual(tuple(outside.iterdir()), ())


if __name__ == "__main__":
    unittest.main()
