from __future__ import annotations

import base64
import os
import re
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
EXPORT_SCRIPT = REPO_ROOT / "tools" / "export_feishu_guides.py"
OUTPUT_PARENT_DIR = "01-中转注册与API密钥"
EXPECTED_OUTPUT_PATHS = {
    f"{OUTPUT_PARENT_DIR}/00-中转注册与API密钥配置教程.md",
    f"{OUTPUT_PARENT_DIR}/01-Codex-API-登录对接教程.md",
    f"{OUTPUT_PARENT_DIR}/02-Claude-Code-配置教程.md",
    f"{OUTPUT_PARENT_DIR}/03-Open-Code-配置教程.md",
    f"{OUTPUT_PARENT_DIR}/04-Open-Claw-配置教程.md",
    f"{OUTPUT_PARENT_DIR}/05-移动端-Chatbox-配置教程.md",
    f"{OUTPUT_PARENT_DIR}/06-Cherry-Studio-图像生成教程.md",
}
PARENT_OUTPUT_PATH = f"{OUTPUT_PARENT_DIR}/00-中转注册与API密钥配置教程.md"
CHILD_OUTPUT_PATHS = tuple(
    sorted(EXPECTED_OUTPUT_PATHS - {PARENT_OUTPUT_PATH})
)
SOURCE_TO_OUTPUT = {
    "registration-key-guide.md": (
        f"{OUTPUT_PARENT_DIR}/00-中转注册与API密钥配置教程.md"
    ),
    "codex-guide.md": f"{OUTPUT_PARENT_DIR}/01-Codex-API-登录对接教程.md",
    "claude-code-guide.md": f"{OUTPUT_PARENT_DIR}/02-Claude-Code-配置教程.md",
    "open-code-guide.md": f"{OUTPUT_PARENT_DIR}/03-Open-Code-配置教程.md",
    "open-claw-guide.md": f"{OUTPUT_PARENT_DIR}/04-Open-Claw-配置教程.md",
    "mobile-guide.md": f"{OUTPUT_PARENT_DIR}/05-移动端-Chatbox-配置教程.md",
    "image-guide.md": f"{OUTPUT_PARENT_DIR}/06-Cherry-Studio-图像生成教程.md",
}
EXPECTED_IMAGE_TARGETS = {
    "registration-key-guide.md": (
        "../../frontend/public/img/codex-guide/image-16.png",
        "../../frontend/public/img/codex-guide/image.png",
        "../../frontend/public/img/codex-guide/image-1.png",
        "../../frontend/public/img/codex-guide/image-17.png",
        "../../frontend/public/img/codex-guide/image-18.png",
        "../../frontend/public/img/codex-guide/image-19.png",
        "../../frontend/public/img/codex-guide/image-20.png",
    ),
    "codex-guide.md": (
        "../../frontend/public/img/codex-guide/image-6.png",
        "../../frontend/public/img/codex-guide/image-7.png",
        "../../frontend/public/img/codex-guide/image-8.png",
        "../../frontend/public/img/codex-guide/image-10.png",
        "../../frontend/public/img/codex-guide/image-11.png",
        "../../frontend/public/img/codex-guide/image-12.png",
        "../../frontend/public/img/codex-guide/image-14.png",
    ),
    "claude-code-guide.md": (
        "../../frontend/public/img/codex-guide/image-22.png",
    ),
    "open-code-guide.md": (),
    "open-claw-guide.md": (),
    "mobile-guide.md": tuple(
        f"../../frontend/public/img/codex-guide/image-{number}.png"
        for number in (32, 33, 34, 35, 36, 37, 40, 42, 44, 45, 46)
    ),
    "image-guide.md": (
        "../../frontend/public/img/image-guide/image.png",
        "../../frontend/public/img/image-guide/image-3.png",
        "../../frontend/public/img/image-guide/image-5.png",
        "../../frontend/public/img/image-guide/image-8.png",
        "../../frontend/public/img/image-guide/image-9.png",
        "../../frontend/public/img/image-guide/image-12.png",
        "../../frontend/public/img/image-guide/image-13.png",
        "../../frontend/public/img/image-guide/image-14.png",
    ),
}
IMAGE_PATTERN = re.compile(r"!\[[^\]]*\]\(([^)\n]+)\)")
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

    def test_source_markdown_matches_the_parent_child_hierarchy(self) -> None:
        source_dir = REPO_ROOT / "docs" / "static-guides"
        parent = (source_dir / "registration-key-guide.md").read_text(
            encoding="utf-8"
        )
        self.assertIn("https://sakai.my/register", parent)
        self.assertIn("https://sakai.my/redeem", parent)
        self.assertIn("创建 API 密钥", parent)

        forbidden_source_markers = (
            "## 页面头部信息",
            "章节快捷入口：",
            "图片目录：",
        )
        forbidden_image_targets = (
            "codex-guide/image-5.png",
            "codex-guide/image-31.png",
            "codex-guide/image-38.png",
            "codex-guide/image-39.png",
            "codex-guide/image-41.png",
            "codex-guide/image-43.png",
            "image-guide/image-6.png",
            "image-guide/image-7.png",
            "image-guide/image-10.png",
            "image-guide/image-11.png",
        )

        for source_name in SOURCE_TO_OUTPUT:
            content = (source_dir / source_name).read_text(encoding="utf-8")
            with self.subTest(source_name=source_name):
                for marker in forbidden_source_markers:
                    self.assertNotIn(marker, content)
                for image_target in forbidden_image_targets:
                    self.assertNotIn(image_target, content)
                if source_name != "registration-key-guide.md":
                    self.assertIn("父教程《中转注册、兑换与 API 密钥配置教程》", content)
                    self.assertNotIn("https://sakai.my/register", content)
                    self.assertNotIn("https://sakai.my/redeem", content)

    def test_unsafe_screenshot_files_cannot_be_published_directly(self) -> None:
        image_root = REPO_ROOT / "frontend" / "public" / "img"
        unsafe_images = (
            "codex-guide/image-5.png",
            "codex-guide/image-31.png",
            "codex-guide/image-38.png",
            "codex-guide/image-39.png",
            "codex-guide/image-41.png",
            "codex-guide/image-43.png",
            "image-guide/image-6.png",
            "image-guide/image-7.png",
            "image-guide/image-10.png",
            "image-guide/image-11.png",
        )

        for relative_path in unsafe_images:
            with self.subTest(relative_path=relative_path):
                self.assertFalse((image_root / relative_path).exists())

    def test_exports_exact_seven_parent_and_child_tutorial_files(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)

            result = self.run_export(output_dir)

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(
                {
                    path.relative_to(output_dir).as_posix()
                    for path in output_dir.rglob("*.md")
                },
                EXPECTED_OUTPUT_PATHS,
            )

    def test_removes_website_only_metadata_and_internal_routes(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            documents = {
                path.name: path.read_text(encoding="utf-8")
                for path in output_dir.rglob("*.md")
            }
            for filename, content in documents.items():
                with self.subTest(filename=filename):
                    for marker in FORBIDDEN_WEBSITE_MARKERS:
                        self.assertFalse(
                            marker in content,
                            f"{filename} 仍包含网页专用内容: {marker}",
                        )

            mobile_guide = documents["05-移动端-Chatbox-配置教程.md"]
            self.assertTrue(
                "父教程《中转注册、兑换与 API 密钥配置教程》" in mobile_guide,
                "移动端子教程未引用父教程",
            )

    def test_separates_registration_parent_from_client_configuration_children(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            parent = (output_dir / PARENT_OUTPUT_PATH).read_text(encoding="utf-8")
            self.assertIn("https://sakai.my/register", parent)
            self.assertIn("https://sakai.my/redeem", parent)
            self.assertIn("创建 API 密钥", parent)
            self.assertIn("### 七、常见问题", parent)
            for child_title in (
                "Codex API 登录对接教程",
                "Claude Code 配置教程",
                "Open Code 配置教程",
                "Open Claw 配置教程",
                "移动端 Chatbox 配置教程",
                "Cherry Studio 图像生成教程",
            ):
                self.assertIn(child_title, parent)

            for child_path in CHILD_OUTPUT_PATHS:
                child = (output_dir / child_path).read_text(encoding="utf-8")
                with self.subTest(child_path=child_path):
                    self.assertIn(
                        "前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》",
                        child,
                    )
                    self.assertNotIn("https://sakai.my/register", child)
                    self.assertNotIn("https://sakai.my/redeem", child)
                    self.assertNotIn("卡密自助购买地址", child)
                    self.assertNotIn("从第一步开始：注册、兑换、创建 Key", child)
                    self.assertNotIn("## 第一章 准备工作", child)
                    self.assertNotIn("## 第二章 创建 API 密钥", child)
                    self.assertNotIn("## 第五章 FAQ", child)

    def test_embeds_every_referenced_png_as_an_exact_data_uri(self) -> None:
        source_dir = REPO_ROOT / "docs" / "static-guides"
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for source_name, output_name in SOURCE_TO_OUTPUT.items():
                source_path = source_dir / source_name
                source_targets = EXPECTED_IMAGE_TARGETS[source_name]
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

            stale_file = output_dir / SOURCE_TO_OUTPUT["codex-guide.md"]
            original_content = stale_file.read_text(encoding="utf-8")
            stale_file.write_text(f"{original_content}\n", encoding="utf-8")
            stale = self.run_export(output_dir, "--check")
            self.assertNotEqual(stale.returncode, 0)
            self.assertIn(stale_file.name, stale.stderr)
            self.assertEqual(stale_file.read_text(encoding="utf-8"), f"{original_content}\n")

            stale_file.write_text(original_content, encoding="utf-8")
            extra_file = output_dir / OUTPUT_PARENT_DIR / "old-guide.md"
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

    def test_exports_are_standalone_well_formed_and_free_of_real_keys(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            total_images = 0
            for output_path in output_dir.rglob("*.md"):
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

            self.assertEqual(total_images, 34)

    def test_exporter_uses_clean_source_markdown_without_content_rewrites(self) -> None:
        from tools import export_feishu_guides as exporter

        source_dir = REPO_ROOT / "docs" / "static-guides"
        for guide in exporter.GUIDES:
            source = (source_dir / guide.source_name).read_text(encoding="utf-8")
            with self.subTest(source_name=guide.source_name):
                self.assertEqual(
                    exporter.render_guide_markdown(guide),
                    source.rstrip() + "\n",
                )

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

    def test_rejects_a_parent_directory_symlink_that_escapes_the_output_directory(self) -> None:
        from tools import export_feishu_guides as exporter

        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            output_dir = root / "exports"
            outside_dir = root / "outside"
            output_dir.mkdir()
            outside_dir.mkdir()
            (output_dir / OUTPUT_PARENT_DIR).symlink_to(
                outside_dir,
                target_is_directory=True,
            )

            with self.assertRaises(ValueError):
                exporter.export_guides(
                    output_dir,
                    ((f"{OUTPUT_PARENT_DIR}/escaped.md", "# escaped\n"),),
                )

            self.assertFalse((outside_dir / "escaped.md").exists())

    def test_preserves_body_headings_code_blocks_tables_and_external_urls(self) -> None:
        required_content = {
            "registration-key-guide.md": (
                "## 中转注册、兑换与 API 密钥配置流程",
                "https://sakai.my/register",
            ),
            "codex-guide.md": (
                "## Codex 客户端配置流程",
                "https://openai.com/zh-Hans-CN/codex/",
            ),
            "claude-code-guide.md": (
                "## Claude Code 配置流程",
                "https://sakai.my/profile",
            ),
            "open-code-guide.md": (
                "## Open Code 配置流程",
                "https://opencode.ai/docs/config",
            ),
            "open-claw-guide.md": (
                "## Open Claw 配置流程",
                "https://sakai.my/profile",
            ),
            "mobile-guide.md": (
                "## Chatbox 配置流程",
                "https://chatboxai.app/zh",
            ),
            "image-guide.md": (
                "## Cherry Studio 图像生成流程",
                "https://cherryai.com.cn/",
            ),
        }
        source_dir = REPO_ROOT / "docs" / "static-guides"
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for source_name, output_name in SOURCE_TO_OUTPUT.items():
                source = (source_dir / source_name).read_text(encoding="utf-8")
                exported = (output_dir / output_name).read_text(encoding="utf-8")
                required_heading, required_url = required_content[source_name]

                with self.subTest(output_name=output_name):
                    self.assertIn(required_heading, exported)
                    self.assertIn(required_url, exported)
                    self.assertEqual(
                        FENCED_CODE_PATTERN.findall(exported),
                        FENCED_CODE_PATTERN.findall(source),
                    )


if __name__ == "__main__":
    unittest.main()
