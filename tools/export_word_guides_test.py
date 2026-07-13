from __future__ import annotations

import hashlib
import io
import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path
from xml.etree import ElementTree


REPO_ROOT = Path(__file__).resolve().parents[1]
EXPORT_SCRIPT = REPO_ROOT / "tools" / "export_word_guides.py"
OUTPUT_PARENT_DIR = "01-中转注册与API密钥"
PARENT_OUTPUT_PATH = f"{OUTPUT_PARENT_DIR}/00-中转注册与API密钥配置教程.docx"
CHILD_OUTPUT_PATHS = (
    f"{OUTPUT_PARENT_DIR}/01-Codex-API-登录对接教程.docx",
    f"{OUTPUT_PARENT_DIR}/02-Claude-Code-配置教程.docx",
    f"{OUTPUT_PARENT_DIR}/03-Open-Code-配置教程.docx",
    f"{OUTPUT_PARENT_DIR}/04-Open-Claw-配置教程.docx",
    f"{OUTPUT_PARENT_DIR}/05-移动端-Chatbox-配置教程.docx",
    f"{OUTPUT_PARENT_DIR}/06-Cherry-Studio-图像生成教程.docx",
)
EXPECTED_OUTPUT_PATHS = {PARENT_OUTPUT_PATH, *CHILD_OUTPUT_PATHS}
EXPECTED_IMAGE_PLACEMENTS = {
    PARENT_OUTPUT_PATH: 6,
    CHILD_OUTPUT_PATHS[0]: 6,
    CHILD_OUTPUT_PATHS[1]: 1,
    CHILD_OUTPUT_PATHS[2]: 0,
    CHILD_OUTPUT_PATHS[3]: 0,
    CHILD_OUTPUT_PATHS[4]: 11,
    CHILD_OUTPUT_PATHS[5]: 8,
}
EXPECTED_UNIQUE_MEDIA = {
    PARENT_OUTPUT_PATH: 6,
    CHILD_OUTPUT_PATHS[0]: 6,
    CHILD_OUTPUT_PATHS[1]: 1,
    CHILD_OUTPUT_PATHS[2]: 0,
    CHILD_OUTPUT_PATHS[3]: 0,
    CHILD_OUTPUT_PATHS[4]: 11,
    CHILD_OUTPUT_PATHS[5]: 8,
}
WORD_NS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
DRAWING_NS = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
REL_NS = "http://schemas.openxmlformats.org/package/2006/relationships"


def document_text(path: Path) -> str:
    with zipfile.ZipFile(path) as archive:
        root = ElementTree.fromstring(archive.read("word/document.xml"))
    paragraphs = []
    for paragraph in root.iter(f"{{{WORD_NS}}}p"):
        paragraphs.append(
            "".join(node.text or "" for node in paragraph.iter(f"{{{WORD_NS}}}t"))
        )
    return "\n".join(paragraphs)


class WordGuideExportTests(unittest.TestCase):
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

    def test_package_font_normalization_preserves_font_names_in_body_text(self) -> None:
        from tools import export_word_guides as exporter

        document_xml = (
            '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            f'<w:document xmlns:w="{WORD_NS}"><w:body><w:p><w:r>'
            '<w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial" '
            'w:eastAsia="PingFang SC" w:cs="PingFang SC"/></w:rPr>'
            '<w:t>正文保留 Arial 与 PingFang SC</w:t>'
            '</w:r></w:p></w:body></w:document>'
        ).encode("utf-8")
        source = io.BytesIO()
        with zipfile.ZipFile(source, "w") as archive:
            archive.writestr("word/document.xml", document_xml)

        normalized = exporter.normalize_docx_package(
            source.getvalue(),
            body_font="Calibri",
            east_asia_font="Hiragino Sans GB",
        )
        with zipfile.ZipFile(io.BytesIO(normalized)) as archive:
            root = ElementTree.fromstring(archive.read("word/document.xml"))

        text = "".join(node.text or "" for node in root.iter(f"{{{WORD_NS}}}t"))
        fonts = next(root.iter(f"{{{WORD_NS}}}rFonts"))
        self.assertEqual(text, "正文保留 Arial 与 PingFang SC")
        self.assertEqual(fonts.attrib[f"{{{WORD_NS}}}ascii"], "Calibri")
        self.assertEqual(fonts.attrib[f"{{{WORD_NS}}}hAnsi"], "Calibri")
        self.assertEqual(fonts.attrib[f"{{{WORD_NS}}}eastAsia"], "Hiragino Sans GB")
        self.assertEqual(fonts.attrib[f"{{{WORD_NS}}}cs"], "Hiragino Sans GB")

    def test_exports_exact_parent_and_six_child_word_files(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)

            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(
                {
                    path.relative_to(output_dir).as_posix()
                    for path in output_dir.rglob("*.docx")
                },
                EXPECTED_OUTPUT_PATHS,
            )

    def test_explicit_v1_edition_matches_the_default_output(self) -> None:
        with tempfile.TemporaryDirectory() as default_directory, tempfile.TemporaryDirectory() as v1_directory:
            default_output = Path(default_directory)
            v1_output = Path(v1_directory)
            default_result = self.run_export(default_output)
            v1_result = self.run_export(v1_output, "--edition", "v1")
            self.assertEqual(default_result.returncode, 0, default_result.stderr)
            self.assertEqual(v1_result.returncode, 0, v1_result.stderr)
            for relative_path in EXPECTED_OUTPUT_PATHS:
                self.assertEqual(
                    (default_output / relative_path).read_bytes(),
                    (v1_output / relative_path).read_bytes(),
                )

    def test_word_packages_embed_all_images_without_external_relationships(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            total_image_placements = 0
            for output_path, expected_placements in EXPECTED_IMAGE_PLACEMENTS.items():
                docx_path = output_dir / output_path
                with zipfile.ZipFile(docx_path) as archive:
                    self.assertIsNone(archive.testzip())
                    document_xml = ElementTree.fromstring(
                        archive.read("word/document.xml")
                    )
                    placements = len(
                        tuple(document_xml.iter(f"{{{DRAWING_NS}}}inline"))
                    )
                    media_paths = {
                        name for name in archive.namelist() if name.startswith("word/media/")
                    }
                    relationships = ElementTree.fromstring(
                        archive.read("word/_rels/document.xml.rels")
                    )
                    external_images = [
                        relation
                        for relation in relationships.iter(f"{{{REL_NS}}}Relationship")
                        if relation.attrib.get("TargetMode") == "External"
                        and relation.attrib.get("Type", "").endswith("/image")
                    ]

                    with self.subTest(output_path=output_path):
                        self.assertEqual(placements, expected_placements)
                        self.assertEqual(
                            len(media_paths),
                            EXPECTED_UNIQUE_MEDIA[output_path],
                        )
                        self.assertFalse(external_images)
                        for media_path in media_paths:
                            self.assertTrue(
                                archive.read(media_path).startswith(b"\x89PNG\r\n\x1a\n")
                            )
                total_image_placements += placements

            self.assertEqual(total_image_placements, 32)

    def test_excludes_outdated_or_unredacted_screenshots(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            forbidden_hashes = {
                "d82ffd2b5a7a3bd8c302849c68592bd7a6fb818573f3b8ccaad6cd8922381962",
                "2e7b40aebdd2fd503d643ae3b3c57bc567679c821ec607b95ed86391957871f3",
                "27bd7b2b9a862b8c705b6a28d9e4d80c24d93bb085f322c1c396638da5d646f6",
                "732cf2ed9b3b58fcbb21c5ffe886c4259b595d6078dbcd367807f18e1c4dbe02",
                "e78a40abc06003630dfdd48725eb2843584daa837cb6fea77087e052ced9aa6f",
                "f7f9dbdc295b63693539484c521a7bbd5ca7037bb8195434111d540bf3a1cc42",
                "704850f01dfb57f7bda2626de8b072248c8b0557c9184d34b3b4ed31004954c8",
                "91ac8e19533f8135bc4eeae9b2ebd097984b051c96e554e40a2f6245dda664e6",
                "f826a908abf27fb922aa7083b65c6c0c7a3dc3a6af60c0c71afd55c60d48458c",
                "61ee0f2073ee41cf836252c7f24dfb6851249caf4b5dde8c06e313311635d170",
            }
            embedded_hashes = set()
            for output_path in EXPECTED_OUTPUT_PATHS:
                with zipfile.ZipFile(output_dir / output_path) as archive:
                    embedded_hashes.update(
                        hashlib.sha256(archive.read(name)).hexdigest()
                        for name in archive.namelist()
                        if name.startswith("word/media/")
                    )

            self.assertTrue(forbidden_hashes.isdisjoint(embedded_hashes))

    def test_parent_contains_registration_and_children_only_contain_client_setup(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            parent = document_text(output_dir / PARENT_OUTPUT_PATH)
            self.assertIn("https://sakai.my/register", parent)
            self.assertIn("https://sakai.my/redeem", parent)
            self.assertIn("创建 API 密钥", parent)
            self.assertIn("七、常见问题", parent)
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
                child = document_text(output_dir / child_path)
                with self.subTest(child_path=child_path):
                    self.assertIn(
                        "前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》",
                        child,
                    )
                    self.assertNotIn("https://sakai.my/register", child)
                    self.assertNotIn("https://sakai.my/redeem", child)
                    self.assertNotIn("卡密自助购买地址", child)
                    self.assertNotIn("从第一步开始：注册、兑换、创建 Key", child)
                    self.assertNotIn("第五章 FAQ", child)

    def test_check_mode_detects_stale_and_extra_word_files_without_writing(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            generated = self.run_export(output_dir)
            self.assertEqual(generated.returncode, 0, generated.stderr)

            current = self.run_export(output_dir, "--check")
            self.assertEqual(current.returncode, 0, current.stderr)

            stale_file = output_dir / CHILD_OUTPUT_PATHS[0]
            original_bytes = stale_file.read_bytes()
            stale_file.write_bytes(original_bytes + b"stale")
            stale = self.run_export(output_dir, "--check")
            self.assertNotEqual(stale.returncode, 0)
            self.assertIn(stale_file.name, stale.stderr)
            self.assertEqual(stale_file.read_bytes(), original_bytes + b"stale")

            stale_file.write_bytes(original_bytes)
            extra_file = output_dir / OUTPUT_PARENT_DIR / "old.docx"
            extra_file.write_bytes(original_bytes)
            extra = self.run_export(output_dir, "--check")
            self.assertNotEqual(extra.returncode, 0)
            self.assertIn(extra_file.name, extra.stderr)
            self.assertTrue(extra_file.exists())

    def test_tables_repeat_headers_prevent_row_splits_and_render_html_breaks(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for output_path in EXPECTED_OUTPUT_PATHS:
                docx_path = output_dir / output_path
                self.assertNotIn("<br>", document_text(docx_path))
                with zipfile.ZipFile(docx_path) as archive:
                    root = ElementTree.fromstring(archive.read("word/document.xml"))
                for table in root.iter(f"{{{WORD_NS}}}tbl"):
                    rows = list(table.findall(f"{{{WORD_NS}}}tr"))
                    self.assertTrue(rows)
                    with self.subTest(output_path=output_path):
                        self.assertIsNotNone(
                            rows[0].find(
                                f"{{{WORD_NS}}}trPr/{{{WORD_NS}}}tblHeader"
                            )
                        )
                        for row in rows:
                            self.assertIsNotNone(
                                row.find(
                                    f"{{{WORD_NS}}}trPr/{{{WORD_NS}}}cantSplit"
                                )
                            )

    def test_uses_a_dedicated_simplified_chinese_font_for_cjk_text(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory)
            result = self.run_export(output_dir)
            self.assertEqual(result.returncode, 0, result.stderr)

            for output_path in EXPECTED_OUTPUT_PATHS:
                with zipfile.ZipFile(output_dir / output_path) as archive:
                    document_xml = archive.read("word/document.xml").decode("utf-8")
                    styles_xml = archive.read("word/styles.xml").decode("utf-8")
                package_xml = document_xml + styles_xml
                with self.subTest(output_path=output_path):
                    self.assertIn('w:eastAsia="PingFang SC"', package_xml)
                    self.assertNotIn("Arial Unicode MS", package_xml)

    def test_rejects_an_output_path_that_escapes_the_output_directory(self) -> None:
        from tools import export_word_guides as exporter

        with tempfile.TemporaryDirectory() as temporary_directory:
            output_dir = Path(temporary_directory) / "exports"
            with self.assertRaises(ValueError):
                exporter.export_documents(output_dir, (("../escaped.docx", b"bad"),))
            self.assertFalse((output_dir.parent / "escaped.docx").exists())

    def test_rejects_a_parent_directory_symlink_that_escapes_the_output_directory(self) -> None:
        from tools import export_word_guides as exporter

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
                exporter.export_documents(
                    output_dir,
                    ((f"{OUTPUT_PARENT_DIR}/escaped.docx", b"bad"),),
                )

            self.assertFalse((outside_dir / "escaped.docx").exists())


if __name__ == "__main__":
    unittest.main()
