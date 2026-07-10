#!/usr/bin/env python3
"""生成可单独上传到飞书的教程 Markdown 文件。"""

from __future__ import annotations

import argparse
import base64
import os
import re
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
SOURCE_DIR = REPO_ROOT / "docs" / "static-guides"
DEFAULT_OUTPUT_DIR = SOURCE_DIR / "feishu"
ALLOWED_IMAGE_DIR = (REPO_ROOT / "frontend" / "public" / "img").resolve()
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
PNG_IHDR_PREFIX = PNG_SIGNATURE + b"\x00\x00\x00\rIHDR"
MINIMUM_PNG_BYTES = 33
MAX_IMAGE_BYTES = 20 * 1024 * 1024
IMAGE_PATTERN = re.compile(r"!\[([^\]]*)\]\(([^)\n]+)\)")


@dataclass(frozen=True)
class GuideSpec:
    source_name: str
    output_name: str
    body_heading: str
    output_title: str


GUIDES = (
    GuideSpec(
        "codex-guide.md",
        "01-Codex-API-登录对接教程.md",
        "## 站点信息卡",
        "Codex API 登录对接教程",
    ),
    GuideSpec(
        "claude-code-guide.md",
        "02-Claude-Code-配置教程.md",
        "## Claude Code 完整接入流程",
        "Claude Code 配置教程",
    ),
    GuideSpec(
        "open-code-guide.md",
        "03-Open-Code-配置教程.md",
        "## Open Code 完整接入流程",
        "Open Code 配置教程",
    ),
    GuideSpec(
        "open-claw-guide.md",
        "04-Open-Claw-配置教程.md",
        "## Open Claw 完整接入流程",
        "Open Claw 配置教程",
    ),
    GuideSpec(
        "mobile-guide.md",
        "05-移动端-Chatbox-配置教程.md",
        "## 移动端完整接入流程",
        "移动端 Chatbox 配置教程",
    ),
    GuideSpec(
        "image-guide.md",
        "06-Cherry-Studio-图像生成教程.md",
        "## Cherry Studio 图像生成完整配置流程",
        "Cherry Studio 图像生成教程",
    ),
)

CLIENT_GUIDE_TABLE_PATTERN = re.compile(
    r"\| 标题 \| 链接 \| 描述 \|\n"
    r"\| --- \| --- \| --- \|\n"
    r"(?:\|[^\n]+\|\n)+"
)
CLIENT_GUIDE_TABLE = """| 教程文档 | 内容 |
| --- | --- |
| Claude Code 配置教程 | 环境变量 / `settings.json` / CLI 验证 |
| Open Code 配置教程 | `opencode.json` / `/connect` 临时切换 |
| Open Claw 配置教程 | 腾讯云在线配置 / 本地配置 |
| 移动端 Chatbox 配置教程 | Chatbox / 手机配置 / 模型切换 |
| 图像生成教程 | Cherry Studio / `imagegen` / `gpt-image-2` |
"""
TEXT_REPLACEMENTS = (
    (
        "在 [Codex 总教程第二章](/codex-guide#chapterKey)",
        "在《Codex API 登录对接教程》第二章",
    ),
    ("独立静态教程页", "独立教程文档"),
    ("打开对应页面后", "阅读对应教程文档后"),
    ("上方独立教程页", "对应的独立教程文档"),
    ("独立教程页", "独立教程文档"),
    ("页面上方交流群二维码", "本文前面的交流群二维码"),
    ("参考页中会直接给出", "“使用密钥”弹窗中会直接给出"),
    (
        "一键查看接入配置，为后续配置做准备，继续往下滑",
        "查看接入配置，为后续配置做准备",
    ),
    ("本页", "本文"),
)


def validated_markdown_name(filename: str) -> str:
    path = Path(filename)
    if path.name != filename or path.suffix.lower() != ".md":
        raise ValueError(f"Markdown 文件名无效: {filename}")
    return filename


def image_path_from_target(source_path: Path, target: str) -> Path:
    image_path = (source_path.parent / target).resolve()
    try:
        image_path.relative_to(ALLOWED_IMAGE_DIR)
    except ValueError as error:
        raise ValueError(f"图片超出允许目录: {target}") from error
    if not image_path.is_file():
        raise ValueError(f"图片文件不存在: {target}")
    return image_path


def image_data_uri(source_path: Path, target: str) -> str:
    image_path = image_path_from_target(source_path, target)
    if image_path.stat().st_size > MAX_IMAGE_BYTES:
        raise ValueError(f"单张图片超过 20 MiB: {target}")
    image_bytes = image_path.read_bytes()
    if len(image_bytes) < MINIMUM_PNG_BYTES or not image_bytes.startswith(PNG_IHDR_PREFIX):
        raise ValueError(f"仅支持结构有效的 PNG 图片: {target}")
    width = int.from_bytes(image_bytes[16:20], byteorder="big")
    height = int.from_bytes(image_bytes[20:24], byteorder="big")
    if width < 1 or height < 1:
        raise ValueError(f"PNG 尺寸无效: {target}")
    if len(image_bytes) > MAX_IMAGE_BYTES:
        raise ValueError(f"单张图片超过 20 MiB: {target}")
    encoded_image = base64.b64encode(image_bytes).decode("ascii")
    return f"data:image/png;base64,{encoded_image}"


def embed_images(markdown: str, source_path: Path) -> str:
    inline_images = tuple(IMAGE_PATTERN.finditer(markdown))
    if markdown.count("![") != len(inline_images):
        raise ValueError(f"不支持引用式或复杂 Markdown 图片: {source_path.name}")
    if re.search(r"<img\b", markdown, flags=re.IGNORECASE):
        raise ValueError(f"不支持 HTML 图片: {source_path.name}")

    def replace_image(match: re.Match[str]) -> str:
        alt_text, target = match.groups()
        return f"![{alt_text}]({image_data_uri(source_path, target)})"

    embedded = IMAGE_PATTERN.sub(replace_image, markdown)
    if any(
        not target.startswith("data:image/png;base64,")
        for _, target in IMAGE_PATTERN.findall(embedded)
    ):
        raise ValueError(f"图片未完全内嵌: {source_path.name}")
    return embedded


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=DEFAULT_OUTPUT_DIR,
        help="输出目录（默认：docs/static-guides/feishu）",
    )
    parser.add_argument(
        "--check",
        action="store_true",
        help="只校验现有产物是否与当前源稿一致",
    )
    return parser.parse_args()


def extract_section(source: str, start_marker: str, end_marker: str) -> str:
    _, start_separator, remainder = source.partition(start_marker)
    section, end_separator, _ = remainder.partition(end_marker)
    if not start_separator or not end_separator:
        raise ValueError(f"源稿缺少区段: {start_marker.strip()}")
    return section.strip()


def render_introduction(source: str, guide: GuideSpec) -> str:
    base_url_match = re.search(r"API base_url：`([^`]+)`", source)
    if base_url_match is None:
        raise ValueError(f"源稿缺少 API base_url: {guide.source_name}")
    lead = extract_section(source, "引导文案：\n\n", "\n\n教程要点：")
    tutorial_points = extract_section(
        source,
        "教程要点：\n\n",
        "\n\n章节快捷入口：",
    )
    return (
        f"# {guide.output_title}\n\n"
        f"> API base_url：`{base_url_match.group(1)}`\n\n"
        f"{lead}\n\n## 教程要点\n\n{tutorial_points}\n\n"
    )


def rewrite_for_feishu(markdown: str) -> str:
    rewritten = CLIENT_GUIDE_TABLE_PATTERN.sub(CLIENT_GUIDE_TABLE, markdown)
    for old_text, new_text in TEXT_REPLACEMENTS:
        rewritten = rewritten.replace(old_text, new_text)
    return rewritten


def render_guide(guide: GuideSpec) -> str:
    source_path = SOURCE_DIR / validated_markdown_name(guide.source_name)
    source = source_path.read_text(encoding="utf-8")
    heading_offset = source.find(guide.body_heading)
    if heading_offset < 0:
        raise ValueError(f"找不到教程正文起始标题: {guide.body_heading}")
    body = source[heading_offset:]
    introduction = render_introduction(source, guide)
    rendered_markdown = rewrite_for_feishu(f"{introduction}{body.rstrip()}\n")
    return embed_images(rendered_markdown, source_path)


def rendered_guides() -> tuple[tuple[str, str], ...]:
    return tuple((guide.output_name, render_guide(guide)) for guide in GUIDES)


def atomic_write_text(destination: Path, content: str) -> None:
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            newline="\n",
            dir=destination.parent,
            prefix=f".{destination.name}.",
            suffix=".tmp",
            delete=False,
        ) as temporary_file:
            temporary_path = Path(temporary_file.name)
            temporary_file.write(content)
            temporary_file.flush()
            os.fsync(temporary_file.fileno())
        os.replace(temporary_path, destination)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()


def export_guides(output_dir: Path, exports: tuple[tuple[str, str], ...]) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    for output_name, content in exports:
        destination = output_dir / validated_markdown_name(output_name)
        atomic_write_text(destination, content)


def check_guides(output_dir: Path, exports: tuple[tuple[str, str], ...]) -> list[str]:
    errors: list[str] = []
    expected_names = {output_name for output_name, _ in exports}
    existing_names = {path.name for path in output_dir.glob("*.md")}
    for missing_name in sorted(expected_names - existing_names):
        errors.append(f"缺少生成文件: {missing_name}")
    for extra_name in sorted(existing_names - expected_names):
        errors.append(f"存在多余生成文件: {extra_name}")
    for output_name, expected_content in exports:
        output_path = output_dir / validated_markdown_name(output_name)
        if output_path.is_symlink() or (output_path.exists() and not output_path.is_file()):
            errors.append(f"生成路径不是普通文件: {output_name}")
            continue
        if output_path.is_file() and output_path.read_bytes() != expected_content.encode("utf-8"):
            errors.append(f"生成文件已过期: {output_name}")
    return errors


def main() -> int:
    args = parse_args()
    output_dir = args.output_dir.resolve()
    try:
        exports = rendered_guides()
        if args.check:
            errors = check_guides(output_dir, exports)
            if errors:
                print("\n".join(errors), file=sys.stderr)
                return 1
            print(f"校验通过：{len(exports)} 份飞书 Markdown 与源稿一致。")
            return 0
        export_guides(output_dir, exports)
    except (OSError, UnicodeError, ValueError) as error:
        print(f"生成失败：{error}", file=sys.stderr)
        return 1
    print(f"已生成 {len(exports)} 份飞书 Markdown：{output_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
