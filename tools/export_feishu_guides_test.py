from __future__ import annotations

import base64
import os
import re
import subprocess
import sys
import tempfile
import unittest
from collections import Counter
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
EXPORT_SCRIPT = REPO_ROOT / "tools" / "export_feishu_guides.py"
EXPECTED_FILENAMES = {
    "01-Codex-API-登录对接教程.md",
    "02-Claude-Code-配置教程.md",
    "03-Open-Code-配置教程.md",
    "04-Open-Claw-配置教程.md",
    "05-移动端-Chatbox-配置教程.md",
    "06-Cherry-Studio-图像生成教程.md",
}
SOURCE_TO_OUTPUT = {
    "codex-guide.md": "01-Codex-API-登录对接教程.md",
    "claude-code-guide.md": "02-Claude-Code-配置教程.md",
    "open-code-guide.md": "03-Open-Code-配置教程.md",
    "open-claw-guide.md": "04-Open-Claw-配置教程.md",
    "mobile-guide.md": "05-移动端-Chatbox-配置教程.md",
    "image-guide.md": "06-Cherry-Studio-图像生成教程.md",
}
SOURCE_BODY_HEADINGS = {
    "codex-guide.md": "## 站点信息卡",
    "claude-code-guide.md": "## Claude Code 完整接入流程",
    "open-code-guide.md": "## Open Code 完整接入流程",
    "open-claw-guide.md": "## Open Claw 完整接入流程",
    "mobile-guide.md": "## 移动端完整接入流程",
    "image-guide.md": "## Cherry Studio 图像生成完整配置流程",
}
IMAGE_PATTERN = re.compile(r"!\[[^\]]*\]\(([^)\n]+)\)")
URL_PATTERN = re.compile(r"https?://[^\s)>`\"']+")
FENCED_CODE_PATTERN = re.compile(r"```[^\n]*\n.*?\n```", re.DOTALL)
FORBIDDEN_WEBSITE_MARKERS = (
    "源页面：",
    "对应文件：",
    "公共外壳：",
    "图片目录：",
    "## 页面头部信息",
    "章节快捷入口：",
    "独立教程页",
    "继续往下滑",
    "参考页中",
    "](/codex-guide",
    "`/claude-code-guide`",
    "`/open-code-guide`",
    "`/open-claw-guide`",
    "`/mobile-guide`",
    "`/image-guide`",
)


class FeishuGuideExportTests(unittest.TestCase):
    def run_export(self, output_dir: Path, *extra_args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                sys.executable,
                str(EXPORT_SCRIPT),
                "--output-dir",
                str(output_dir),
                *extra_args,
            ],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_exports_exact_six_tutorial_files(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)

            result = self.run_export(output_dir)

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(
                {path.name for path in output_dir.glob("*.md")},
                EXPECTED_FILENAMES,
            )

    def test_removes_website_only_metadata_and_internal_routes(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            documents = {
                path.name: path.read_text(encoding="utf-8")
                for path in output_dir.glob("*.md")
            }
            for filename, content in documents.items():
                with self.subTest(filename=filename):
                    for marker in FORBIDDEN_WEBSITE_MARKERS:
                        self.assertFalse(
                            marker in content,
                            f"{filename} 仍包含网页专用内容: {marker}",
                        )

            codex_guide = documents["01-Codex-API-登录对接教程.md"]
            self.assertTrue(
                "图像生成教程" in codex_guide,
                "Codex 教程未包含图像生成教程入口",
            )
            mobile_guide = documents["05-移动端-Chatbox-配置教程.md"]
            self.assertTrue(
                "《Codex API 登录对接教程》第二章" in mobile_guide,
                "移动端教程未将网站锚点改为文档章节引用",
            )

    def test_embeds_every_referenced_png_as_an_exact_data_uri(self) -> None:
        source_dir = REPO_ROOT / "docs" / "static-guides"
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for source_name, output_name in SOURCE_TO_OUTPUT.items():
                source_path = source_dir / source_name
                source_targets = IMAGE_PATTERN.findall(
                    source_path.read_text(encoding="utf-8")
                )
                output_targets = IMAGE_PATTERN.findall(
                    (output_dir / output_name).read_text(encoding="utf-8")
                )
                with self.subTest(output_name=output_name):
                    self.assertEqual(len(output_targets), len(source_targets))
                    for source_target, output_target in zip(
                        source_targets,
                        output_targets,
                    ):
                        self.assertTrue(
                            output_target.startswith("data:image/png;base64,"),
                            output_target[:120],
                        )
                        encoded_image = output_target.split(",", 1)[1]
                        decoded_image = base64.b64decode(
                            encoded_image,
                            validate=True,
                        )
                        expected_image = (source_path.parent / source_target).read_bytes()
                        self.assertEqual(decoded_image, expected_image)
                        self.assertTrue(decoded_image.startswith(b"\x89PNG\r\n\x1a\n"))

    def test_check_mode_detects_stale_or_extra_exports_without_writing(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            generated = self.run_export(output_dir)
            self.assertEqual(generated.returncode, 0, generated.stderr)

            current = self.run_export(output_dir, "--check")
            self.assertEqual(current.returncode, 0, current.stderr)

            stale_file = output_dir / "01-Codex-API-登录对接教程.md"
            original_content = stale_file.read_text(encoding="utf-8")
            stale_file.write_text(f"{original_content}\n", encoding="utf-8")
            stale = self.run_export(output_dir, "--check")
            self.assertNotEqual(stale.returncode, 0)
            self.assertIn(stale_file.name, stale.stderr)
            self.assertEqual(stale_file.read_text(encoding="utf-8"), f"{original_content}\n")

            stale_file.write_text(original_content, encoding="utf-8")
            extra_file = output_dir / "old-guide.md"
            extra_file.write_text("# old\n", encoding="utf-8")
            extra = self.run_export(output_dir, "--check")
            self.assertNotEqual(extra.returncode, 0)
            self.assertIn(extra_file.name, extra.stderr)
            self.assertTrue(extra_file.exists())

            extra_file.unlink()
            stale_file.unlink()
            stale_file.mkdir()
            wrong_type = self.run_export(output_dir, "--check")
            self.assertNotEqual(wrong_type.returncode, 0)
            self.assertIn(stale_file.name, wrong_type.stderr)
            self.assertTrue(stale_file.is_dir())

            stale_file.rmdir()
            regenerated = self.run_export(output_dir)
            self.assertEqual(regenerated.returncode, 0, regenerated.stderr)
            lf_bytes = stale_file.read_bytes()
            stale_file.write_bytes(lf_bytes.replace(b"\n", b"\r\n"))
            crlf_drift = self.run_export(output_dir, "--check")
            self.assertNotEqual(crlf_drift.returncode, 0)
            self.assertIn(stale_file.name, crlf_drift.stderr)

    def test_codex_source_tracks_the_current_image_guide_navigation(self) -> None:
        codex_source = (
            REPO_ROOT / "docs" / "static-guides" / "codex-guide.md"
        ).read_text(encoding="utf-8")

        self.assertIn(
            "Claude Code、Open Code、Open Claw、移动端和图像生成请打开对应独立教程页。",
            codex_source,
        )
        self.assertIn("- 图像生成", codex_source)
        self.assertEqual(
            codex_source.count("| 图像生成教程 | `/image-guide` |"),
            2,
        )

    def test_exports_are_standalone_well_formed_and_free_of_real_keys(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            total_images = 0
            for output_path in output_dir.glob("*.md"):
                content = output_path.read_text(encoding="utf-8")
                image_targets = IMAGE_PATTERN.findall(content)
                total_images += len(image_targets)
                with self.subTest(output_name=output_path.name):
                    self.assertTrue(content.endswith("\n"))
                    self.assertFalse(content.startswith("\ufeff"))
                    self.assertEqual(len(re.findall(r"^# ", content, re.MULTILINE)), 1)
                    self.assertEqual(content.count("```") % 2, 0)
                    self.assertLess(output_path.stat().st_size, 20 * 1024 * 1024)
                    self.assertNotRegex(content, r"sk-[A-Za-z0-9_-]{16,}")
                    self.assertNotIn("frontend/public/img/", content)
                    for image_target in image_targets:
                        self.assertTrue(image_target.startswith("data:image/png;base64,"))

            self.assertEqual(total_images, 45)

    def test_keeps_source_leads_and_tutorial_points_in_the_export_intro(self) -> None:
        source_dir = REPO_ROOT / "docs" / "static-guides"
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for source_name, output_name in SOURCE_TO_OUTPUT.items():
                source = (source_dir / source_name).read_text(encoding="utf-8")
                lead = source.split("引导文案：\n\n", 1)[1].split(
                    "\n\n教程要点：",
                    1,
                )[0]
                points_block = source.split("教程要点：\n\n", 1)[1].split(
                    "\n\n章节快捷入口：",
                    1,
                )[0]
                tutorial_points = [
                    line for line in points_block.splitlines() if line.startswith("- ")
                ]
                exported = (output_dir / output_name).read_text(encoding="utf-8")
                expected_lead = lead.replace("独立教程页", "独立教程文档")
                with self.subTest(output_name=output_name):
                    self.assertTrue(
                        expected_lead in exported,
                        f"{output_name} 未保留源稿引导文案",
                    )
                    self.assertTrue(
                        "## 教程要点\n" in exported,
                        f"{output_name} 缺少教程要点章节",
                    )
                    for tutorial_point in tutorial_points:
                        self.assertTrue(
                            tutorial_point in exported,
                            f"{output_name} 缺少教程要点: {tutorial_point}",
                        )

            mobile_export = (
                output_dir / "05-移动端-Chatbox-配置教程.md"
            ).read_text(encoding="utf-8")
            self.assertNotIn("在 《Codex API 登录对接教程》", mobile_export)

    def test_rejects_unsupported_or_external_image_syntax(self) -> None:
        from tools import export_feishu_guides as exporter

        source_path = REPO_ROOT / "docs" / "static-guides" / "image-guide.md"
        invalid_markdown = (
            "![引用式图片][image-id]\n[image-id]: ../../frontend/public/img/image-guide/image.png\n",
            '<img src="../../frontend/public/img/image-guide/image.png">\n',
            "![远程图片](https://example.com/image.png)\n",
        )

        for markdown in invalid_markdown:
            with self.subTest(markdown=markdown.splitlines()[0]):
                with self.assertRaises(ValueError):
                    exporter.embed_images(markdown, source_path)

    def test_rejects_a_truncated_png(self) -> None:
        from tools import export_feishu_guides as exporter

        image_dir = REPO_ROOT / "frontend" / "public" / "img"
        source_path = REPO_ROOT / "docs" / "static-guides" / "image-guide.md"
        with tempfile.NamedTemporaryFile(
            dir=image_dir,
            prefix=".feishu-export-test-",
            suffix=".png",
            delete=False,
        ) as temporary_image:
            temporary_image.write(b"\x89PNG\r\n\x1a\n")
            image_path = Path(temporary_image.name)
        image_target = os.path.relpath(image_path, start=source_path.parent)

        try:
            with self.assertRaisesRegex(ValueError, "PNG"):
                exporter.image_data_uri(source_path, image_target)
        finally:
            image_path.unlink(missing_ok=True)

    def test_rejects_an_output_name_that_escapes_the_output_directory(self) -> None:
        from tools import export_feishu_guides as exporter

        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            output_dir = root / "exports"
            escaped_path = root / "escaped.md"

            with self.assertRaises(ValueError):
                exporter.export_guides(
                    output_dir,
                    (("../escaped.md", "# escaped\n"),),
                )

            self.assertFalse(escaped_path.exists())

    def test_preserves_body_headings_code_blocks_tables_and_external_urls(self) -> None:
        source_dir = REPO_ROOT / "docs" / "static-guides"
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for source_name, output_name in SOURCE_TO_OUTPUT.items():
                source = (source_dir / source_name).read_text(encoding="utf-8")
                body_heading = SOURCE_BODY_HEADINGS[source_name]
                source_body = source[source.index(body_heading) :]
                exported = (output_dir / output_name).read_text(encoding="utf-8")
                exported_body = exported[exported.index(body_heading) :]

                expected_headings = re.findall(r"^#{2,6} .+$", source_body, re.MULTILINE)
                expected_headings = [
                    heading.replace(
                        "一键查看接入配置，为后续配置做准备，继续往下滑",
                        "查看接入配置，为后续配置做准备",
                    )
                    for heading in expected_headings
                ]
                actual_headings = re.findall(r"^#{2,6} .+$", exported_body, re.MULTILINE)
                source_table_count = len(
                    re.findall(r"^\|(?: --- \|)+$", source_body, re.MULTILINE)
                )
                output_table_count = len(
                    re.findall(r"^\|(?: --- \|)+$", exported_body, re.MULTILINE)
                )

                with self.subTest(output_name=output_name):
                    self.assertEqual(actual_headings, expected_headings)
                    self.assertEqual(
                        FENCED_CODE_PATTERN.findall(exported),
                        FENCED_CODE_PATTERN.findall(source),
                    )
                    self.assertEqual(
                        Counter(URL_PATTERN.findall(exported)),
                        Counter(URL_PATTERN.findall(source)),
                    )
                    self.assertEqual(output_table_count, source_table_count)


if __name__ == "__main__":
    unittest.main()
