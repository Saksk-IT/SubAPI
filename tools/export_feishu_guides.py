#!/usr/bin/env python3
"""将整理后的教程 Markdown 导出为图片内嵌的飞书上传文件。"""

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
OUTPUT_PARENT_DIR = "01-中转注册与API密钥"


@dataclass(frozen=True)
class GuideSpec:
    source_name: str
    output_name: str


GUIDES = (
    GuideSpec(
        "registration-key-guide.md",
        f"{OUTPUT_PARENT_DIR}/00-中转注册与API密钥配置教程.md",
    ),
    GuideSpec(
        "codex-guide.md",
        f"{OUTPUT_PARENT_DIR}/01-Codex-API-登录对接教程.md",
    ),
    GuideSpec(
        "claude-code-guide.md",
        f"{OUTPUT_PARENT_DIR}/02-Claude-Code-配置教程.md",
    ),
    GuideSpec(
        "open-code-guide.md",
        f"{OUTPUT_PARENT_DIR}/03-Open-Code-配置教程.md",
    ),
    GuideSpec(
        "open-claw-guide.md",
        f"{OUTPUT_PARENT_DIR}/04-Open-Claw-配置教程.md",
    ),
    GuideSpec(
        "mobile-guide.md",
        f"{OUTPUT_PARENT_DIR}/05-移动端-Chatbox-配置教程.md",
    ),
    GuideSpec(
        "image-guide.md",
        f"{OUTPUT_PARENT_DIR}/06-Cherry-Studio-图像生成教程.md",
    ),
)


def validated_markdown_name(filename: str) -> str:
    path = Path(filename)
    if path.name != filename or path.suffix.lower() != ".md":
        raise ValueError(f"Markdown 文件名无效: {filename}")
    return filename


def validated_markdown_path(relative_path: str) -> Path:
    path = Path(relative_path)
    if (
        path.is_absolute()
        or path.suffix.lower() != ".md"
        or not path.parts
        or any(part in {"", ".", ".."} for part in path.parts)
    ):
        raise ValueError(f"Markdown 输出路径无效: {relative_path}")
    return path


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
    if len(image_bytes) < MINIMUM_PNG_BYTES or not image_bytes.startswith(
        PNG_IHDR_PREFIX
    ):
        raise ValueError(f"仅支持结构有效的 PNG 图片: {target}")
    width = int.from_bytes(image_bytes[16:20], byteorder="big")
    height = int.from_bytes(image_bytes[20:24], byteorder="big")
    if width < 1 or height < 1:
        raise ValueError(f"PNG 尺寸无效: {target}")
    return (
        "data:image/png;base64,"
        + base64.b64encode(image_bytes).decode("ascii")
    )


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


def render_guide_markdown(guide: GuideSpec) -> str:
    source_path = SOURCE_DIR / validated_markdown_name(guide.source_name)
    source = source_path.read_text(encoding="utf-8")
    if source.startswith("\ufeff"):
        raise ValueError(f"Markdown 源稿包含 BOM: {guide.source_name}")
    if len(re.findall(r"^# ", source, flags=re.MULTILINE)) != 1:
        raise ValueError(f"Markdown 源稿必须且只能包含一个一级标题: {guide.source_name}")
    if source.count("```") % 2:
        raise ValueError(f"Markdown 源稿代码块未闭合: {guide.source_name}")
    return source.rstrip() + "\n"


def render_guide(guide: GuideSpec) -> str:
    source_path = SOURCE_DIR / validated_markdown_name(guide.source_name)
    return embed_images(render_guide_markdown(guide), source_path)


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


def safe_output_destination(output_dir: Path, relative_path: Path) -> Path:
    output_root = output_dir.resolve()
    destination = output_root / relative_path
    destination.parent.mkdir(parents=True, exist_ok=True)
    resolved_parent = destination.parent.resolve()
    try:
        resolved_parent.relative_to(output_root)
    except ValueError as error:
        raise ValueError(f"Markdown 输出目录超出允许范围: {relative_path}") from error
    return resolved_parent / destination.name


def export_guides(output_dir: Path, exports: tuple[tuple[str, str], ...]) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    for output_name, content in exports:
        destination = safe_output_destination(
            output_dir,
            validated_markdown_path(output_name),
        )
        atomic_write_text(destination, content)


def check_guides(output_dir: Path, exports: tuple[tuple[str, str], ...]) -> list[str]:
    errors: list[str] = []
    expected_names = {output_name for output_name, _ in exports}
    existing_names = {
        path.relative_to(output_dir).as_posix()
        for path in output_dir.rglob("*.md")
    }
    for missing_name in sorted(expected_names - existing_names):
        errors.append(f"缺少生成文件: {missing_name}")
    for extra_name in sorted(existing_names - expected_names):
        errors.append(f"存在多余生成文件: {extra_name}")
    for output_name, expected_content in exports:
        output_path = output_dir / validated_markdown_path(output_name)
        if output_path.is_symlink() or (
            output_path.exists() and not output_path.is_file()
        ):
            errors.append(f"生成路径不是普通文件: {output_name}")
            continue
        if output_path.is_file() and (
            output_path.read_bytes() != expected_content.encode("utf-8")
        ):
            errors.append(f"生成文件已过期: {output_name}")
    return errors


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
