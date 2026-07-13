#!/usr/bin/env python3
"""V2 教程导出的共享内容、媒体、安全路径与原子写入能力。"""

from __future__ import annotations

import base64
import hashlib
import io
import json
import os
import re
import shutil
import tempfile
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Iterable

REPO_ROOT = Path(__file__).resolve().parents[1]
CONTENT_DIR = REPO_ROOT / "frontend" / "src" / "content" / "guides-v2"
MANIFEST_PATH = CONTENT_DIR / "manifest.generated.json"
MEDIA_MANIFEST_PATH = CONTENT_DIR / "media-manifest.json"
DEFAULT_MARKDOWN_OUTPUT_DIR = (
    REPO_ROOT / "docs" / "static-guides" / "feishu-v2"
)
DEFAULT_WORD_OUTPUT_DIR = (
    REPO_ROOT / "docs" / "static-guides" / "feishu-word-v2"
)
OUTPUT_PARENT_DIR = "02-AI客户端使用指南"
BRAND = "AI 客户端使用指南"
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
FRONTMATTER_PATTERN = re.compile(r"\A---\r?\n[\s\S]*?\r?\n---\r?\n?")
IMAGE_PATTERN = re.compile(
    r"!\[([^\]]*)\]\((/img/guides/v2/[^\s)]+\.webp)"
    r'(?:\s+"([^"]*)")?\)'
)
INTERNAL_LINK_PATTERN = re.compile(
    r"\[([^\]]+)\]\(/guides/v2(?:/[a-z0-9-]+)?(?:#[a-z0-9-]+)?\)"
)
HEADING_ANCHOR_PATTERN = re.compile(
    r"\s+\{#[a-z][a-z0-9-]*\}(?=\s*$)",
    re.MULTILINE,
)
VIDEO_PATTERN = re.compile(r"<video\b[^>]*>[\s\S]*?</video>", re.IGNORECASE)
VIDEO_ATTRIBUTE_PATTERN = re.compile(r'\b(src|poster)="([^"]+)"', re.IGNORECASE)
SHA256_PATTERN = re.compile(r"\A[a-f0-9]{64}\Z")


@dataclass(frozen=True)
class GuideSpec:
    source_name: str
    output_stem: str
    slug: str

    def output_name(self, suffix: str) -> str:
        return f"{OUTPUT_PARENT_DIR}/{self.output_stem}{suffix}"


@dataclass(frozen=True)
class MediaSpec:
    web_path: str
    export_path: Path
    alt: str
    caption: str
    kind: str
    poster_path: str | None


GUIDES = (
    GuideSpec("index.md", "00-教程中心", "index"),
    GuideSpec("get-started.md", "01-快速开始", "get-started"),
    GuideSpec("codex.md", "02-Codex", "codex"),
    GuideSpec("claude-code.md", "03-Claude-Code", "claude-code"),
    GuideSpec("opencode.md", "04-OpenCode", "opencode"),
    GuideSpec("openclaw.md", "05-OpenClaw", "openclaw"),
    GuideSpec("chatbox-mobile.md", "06-Chatbox-移动端", "chatbox-mobile"),
    GuideSpec(
        "cherry-studio-image.md",
        "07-Cherry-Studio-生图",
        "cherry-studio-image",
    ),
    GuideSpec("troubleshooting.md", "08-公共排错中心", "troubleshooting"),
)


def _safe_regular_file(path: Path, allowed_root: Path, label: str) -> Path:
    if path.is_symlink():
        raise ValueError(f"{label}拒绝符号链接: {path}")
    if not path.is_file():
        raise ValueError(f"{label}必须是普通文件: {path}")
    resolved = path.resolve()
    try:
        resolved.relative_to(allowed_root.resolve())
    except ValueError as error:
        raise ValueError(f"{label}真实路径超出允许目录: {path}") from error
    return resolved


def _load_json(path: Path, allowed_root: Path, label: str) -> dict:
    safe_path = _safe_regular_file(path, allowed_root, label)
    value = json.loads(safe_path.read_text("utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{label}必须是 JSON 对象")
    return value


def split_frontmatter(source: str, source_name: str) -> str:
    match = FRONTMATTER_PATTERN.match(source)
    if match is None:
        raise ValueError(f"V2 源稿缺少有效 Frontmatter: {source_name}")
    body = source[match.end() :]
    if not body.startswith("# "):
        raise ValueError(f"V2 正文必须以一级标题开始: {source_name}")
    return body


def _manifest_hashes(content_dir: Path) -> dict[str, str]:
    manifest = _load_json(
        content_dir / "manifest.generated.json",
        content_dir,
        "V2 内容 manifest",
    )
    entries = manifest.get("entries")
    if manifest.get("version") != "v2" or not isinstance(entries, list):
        raise ValueError("V2 内容 manifest 结构无效")
    hashes: dict[str, str] = {}
    for entry in entries:
        if not isinstance(entry, dict):
            raise ValueError("V2 内容 manifest entry 无效")
        source = entry.get("source")
        digest = entry.get("contentHash")
        if (
            not isinstance(source, str)
            or not isinstance(digest, str)
            or not SHA256_PATTERN.match(digest)
            or source in hashes
        ):
            raise ValueError("V2 内容 manifest source/hash 无效")
        hashes[source] = digest
    expected = {guide.source_name for guide in GUIDES}
    if set(hashes) != expected:
        raise ValueError("V2 内容 manifest 必须精确包含 9 份教程")
    return hashes


def load_manifest_metadata(content_dir: Path = CONTENT_DIR) -> dict[str, dict]:
    manifest = _load_json(
        content_dir / "manifest.generated.json",
        content_dir,
        "V2 内容 manifest",
    )
    entries = manifest.get("entries")
    if manifest.get("version") != "v2" or not isinstance(entries, list):
        raise ValueError("V2 内容 manifest 结构无效")
    metadata: dict[str, dict] = {}
    for entry in entries:
        if not isinstance(entry, dict) or not isinstance(entry.get("meta"), dict):
            raise ValueError("V2 内容 manifest meta 无效")
        source = entry.get("source")
        if not isinstance(source, str) or source in metadata:
            raise ValueError("V2 内容 manifest source 无效")
        metadata[source] = dict(entry["meta"])
    if set(metadata) != {guide.source_name for guide in GUIDES}:
        raise ValueError("V2 内容 manifest 必须精确包含 9 份教程元数据")
    return metadata


def load_verified_bodies(content_dir: Path = CONTENT_DIR) -> dict[str, str]:
    if content_dir.is_symlink() or not content_dir.is_dir():
        raise ValueError(f"V2 内容目录无效或为符号链接: {content_dir}")
    hashes = _manifest_hashes(content_dir)
    bodies: dict[str, str] = {}
    for guide in GUIDES:
        source_path = _safe_regular_file(
            content_dir / guide.source_name,
            content_dir,
            "V2 Markdown ",
        )
        source = source_path.read_text("utf-8")
        body = split_frontmatter(source, guide.source_name)
        actual_hash = hashlib.sha256(body.encode("utf-8")).hexdigest()
        if actual_hash != hashes[guide.source_name]:
            raise ValueError(
                f"{guide.source_name} 与 manifest.generated.json 哈希不一致；"
                "请先运行 pnpm --dir frontend guides:v2:manifest"
            )
        bodies[guide.source_name] = body
    return bodies


def _validated_repo_relative(path_value: str, label: str) -> Path:
    path = PurePosixPath(path_value)
    if (
        path.is_absolute()
        or not path.parts
        or any(part in {"", ".", ".."} for part in path.parts)
        or re.match(r"^[a-zA-Z]:", path_value)
        or path_value.startswith(("\\", "//"))
    ):
        raise ValueError(f"{label}路径无效: {path_value}")
    return Path(*path.parts)


def _png_data_uri(path: Path, expected_hash: str) -> str:
    try:
        from PIL import Image as PillowImage
        from PIL import UnidentifiedImageError
    except ImportError as error:
        raise ValueError(
            "V2 PNG 校验需要 Pillow；请安装 Pillow 后重试。"
        ) from error

    image_bytes = path.read_bytes()
    if hashlib.sha256(image_bytes).hexdigest() != expected_hash:
        raise ValueError(f"V2 导出图片哈希不匹配: {path}")
    try:
        with PillowImage.open(io.BytesIO(image_bytes)) as image:
            if image.format != "PNG":
                raise ValueError(f"V2 导出图片不是有效 PNG: {path}")
            image.verify()
    except (UnidentifiedImageError, OSError, SyntaxError) as error:
        raise ValueError(f"V2 导出图片不是有效 PNG: {path}") from error
    return "data:image/png;base64," + base64.b64encode(image_bytes).decode("ascii")


def load_media(content_dir: Path = CONTENT_DIR) -> dict[str, tuple[MediaSpec, str]]:
    manifest = _load_json(
        content_dir / "media-manifest.json",
        content_dir,
        "V2 媒体 manifest",
    )
    entries = manifest.get("media")
    if manifest.get("version") != "v2" or not isinstance(entries, list):
        raise ValueError("V2 媒体 manifest 结构无效")
    media: dict[str, tuple[MediaSpec, str]] = {}
    for entry in entries:
        if not isinstance(entry, dict):
            raise ValueError("V2 媒体 manifest entry 无效")
        web_path = entry.get("webPath")
        export_path = entry.get("exportPath")
        png_sha256 = entry.get("pngSha256")
        if not isinstance(web_path, str) or not isinstance(export_path, str):
            raise ValueError("V2 媒体路径缺失")
        if not isinstance(png_sha256, str) or not SHA256_PATTERN.fullmatch(
            png_sha256
        ):
            raise ValueError(f"V2 媒体 pngSha256 无效: {web_path}")
        if web_path in media:
            raise ValueError(f"V2 媒体 webPath 重复: {web_path}")
        relative = _validated_repo_relative(export_path, "V2 媒体 exportPath")
        image_path = _safe_regular_file(
            REPO_ROOT / relative,
            REPO_ROOT / "frontend" / "public" / "img" / "guides" / "v2",
            "V2 导出图片 ",
        )
        spec = MediaSpec(
            web_path=web_path,
            export_path=image_path,
            alt=str(entry.get("alt", "")),
            caption=str(entry.get("caption", "")),
            kind=str(entry.get("kind", "")),
            poster_path=(
                str(entry["posterPath"])
                if isinstance(entry.get("posterPath"), str)
                else None
            ),
        )
        media[web_path] = (spec, _png_data_uri(image_path, png_sha256))
    return media


def degrade_video(
    html: str,
    *,
    poster_data_uri: str,
    alt: str,
    caption: str,
    link: str,
) -> str:
    replacement = (
        f"![{alt}]({poster_data_uri})\n\n"
        f"视频说明：{caption}\n\n"
        f"视频链接：{link}"
    )
    return VIDEO_PATTERN.sub(replacement, html)


def _normalize_callouts(markdown: str) -> str:
    labels = {
        "NOTE": "说明",
        "TIP": "提示",
        "WARNING": "注意",
        "SUCCESS": "完成",
    }
    lines = markdown.splitlines()
    output: list[str] = []
    index = 0
    while index < len(lines):
        marker = re.fullmatch(r">\s*\[!([A-Z]+)\]\s*", lines[index])
        if marker and index + 1 < len(lines) and lines[index + 1].startswith(">"):
            label = labels.get(marker.group(1), "说明")
            output.append(f"> {label}：{lines[index + 1].lstrip('> ').strip()}")
            index += 2
            continue
        output.append(lines[index])
        index += 1
    return "\n".join(output)


def render_markdown(
    body: str,
    media: dict[str, tuple[MediaSpec, str]],
) -> str:
    def replace_video(match: re.Match[str]) -> str:
        attributes = {
            name.lower(): value
            for name, value in VIDEO_ATTRIBUTE_PATTERN.findall(match.group(0))
        }
        source = attributes.get("src", "")
        if source not in media:
            raise ValueError(f"V2 视频未在 media-manifest.json 登记: {source}")
        spec, poster_data_uri = media[source]
        if spec.kind != "video":
            raise ValueError(f"V2 媒体不是视频类型: {source}")
        return degrade_video(
            match.group(0),
            poster_data_uri=poster_data_uri,
            alt=spec.alt or "教程视频封面",
            caption=spec.caption or "请按视频步骤完成配置。",
            link=source,
        )

    body = VIDEO_PATTERN.sub(replace_video, body)

    def replace_image(match: re.Match[str]) -> str:
        alt, web_path, title = match.groups()
        if web_path not in media:
            raise ValueError(f"V2 图片未在 media-manifest.json 登记: {web_path}")
        spec, data_uri = media[web_path]
        caption = title or spec.caption
        rendered_alt = alt or spec.alt
        caption_line = f"\n\n图：{caption}" if caption else ""
        return f"![{rendered_alt}]({data_uri}){caption_line}"

    rendered = IMAGE_PATTERN.sub(replace_image, body)
    if "/img/guides/v2/" in rendered:
        raise ValueError("V2 图片未全部转换为 PNG 内嵌数据")
    rendered = INTERNAL_LINK_PATTERN.sub(r"\1", rendered)
    rendered = HEADING_ANCHOR_PATTERN.sub("", rendered)
    rendered = _normalize_callouts(rendered)
    return rendered.rstrip() + "\n"


def rendered_markdown_exports() -> tuple[tuple[str, bytes], ...]:
    bodies = load_verified_bodies()
    media = load_media()
    return tuple(
        (
            guide.output_name(".md"),
            render_markdown(bodies[guide.source_name], media).encode("utf-8"),
        )
        for guide in GUIDES
    )


def validated_output_path(relative_path: str) -> Path:
    path = PurePosixPath(relative_path)
    if (
        path.is_absolute()
        or not path.parts
        or any(part in {"", ".", ".."} for part in path.parts)
        or re.match(r"^[a-zA-Z]:", relative_path)
        or relative_path.startswith(("\\", "//"))
    ):
        raise ValueError(f"导出路径无效: {relative_path}")
    return Path(*path.parts)


def _write_staging_tree(
    staging: Path,
    exports: tuple[tuple[Path, bytes], ...],
) -> None:
    for relative_path, content in exports:
        destination = staging / relative_path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(content)
        destination.chmod(0o644)
    directory_fd = os.open(staging, os.O_RDONLY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)


def atomic_export_tree(
    output_dir: Path,
    exports: Iterable[tuple[str, bytes]],
) -> None:
    normalized = tuple(
        (validated_output_path(name), bytes(content)) for name, content in exports
    )
    if len({path.as_posix() for path, _ in normalized}) != len(normalized):
        raise ValueError("导出路径重复")
    if output_dir.is_symlink():
        raise ValueError(f"导出目录拒绝符号链接: {output_dir}")
    if output_dir.exists() and not output_dir.is_dir():
        raise ValueError(f"导出路径已存在且不是目录: {output_dir}")
    parent = output_dir.parent
    parent.mkdir(parents=True, exist_ok=True)
    staging = Path(tempfile.mkdtemp(prefix=f".{output_dir.name}.staging-", dir=parent))
    backup = parent / f".{output_dir.name}.backup"
    moved_existing = False
    try:
        _write_staging_tree(staging, normalized)
        if backup.exists() or backup.is_symlink():
            raise ValueError(f"导出备份路径已存在: {backup}")
        if output_dir.exists():
            os.replace(output_dir, backup)
            moved_existing = True
        os.replace(staging, output_dir)
        if moved_existing:
            shutil.rmtree(backup)
    except Exception:
        if moved_existing and backup.exists() and not output_dir.exists():
            os.replace(backup, output_dir)
        raise
    finally:
        if staging.exists():
            shutil.rmtree(staging)
        if backup.exists() and output_dir.exists():
            shutil.rmtree(backup)


def check_export_tree(
    output_dir: Path,
    exports: Iterable[tuple[str, bytes]],
) -> list[str]:
    normalized = tuple(
        (validated_output_path(name), bytes(content)) for name, content in exports
    )
    if output_dir.is_symlink() or not output_dir.is_dir():
        return [f"生成目录不存在或不是普通目录: {output_dir}"]
    expected = {path.as_posix() for path, _ in normalized}
    existing = {
        path.relative_to(output_dir).as_posix()
        for path in output_dir.rglob("*")
        if path.is_file() or path.is_symlink()
    }
    errors = [f"缺少生成文件: {name}" for name in sorted(expected - existing)]
    errors.extend(f"存在多余生成文件: {name}" for name in sorted(existing - expected))
    for relative_path, content in normalized:
        destination = output_dir / relative_path
        if destination.is_symlink() or (destination.exists() and not destination.is_file()):
            errors.append(f"生成路径不是普通文件: {relative_path.as_posix()}")
        elif destination.is_file() and destination.read_bytes() != content:
            errors.append(f"生成文件已过期: {relative_path.as_posix()}")
    return errors
