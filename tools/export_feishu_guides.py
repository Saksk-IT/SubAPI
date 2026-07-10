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
OUTPUT_PARENT_DIR = "01-中转注册与API密钥"


@dataclass(frozen=True)
class GuideSpec:
    source_name: str
    output_name: str
    body_heading: str
    output_title: str
    lead_override: str | None = None
    tutorial_points_override: tuple[str, ...] | None = None
    preface: str = ""
    body_end_heading: str | None = None
    removed_sections: tuple[tuple[str, str], ...] = ()
    replacements: tuple[tuple[str, str], ...] = ()


CHILD_GUIDE_LEAD = (
    "前置步骤：请先完成父教程《中转注册、兑换与 API 密钥配置教程》，"
    "准备好自己的 `base_url` 和 API Key。本文只讲客户端配置，"
    "不再重复注册、兑换和创建密钥。"
)


GUIDES = (
    GuideSpec(
        "registration-key-guide.md",
        f"{OUTPUT_PARENT_DIR}/00-中转注册与API密钥配置教程.md",
        "## 中转注册、兑换与 API 密钥配置流程",
        "中转注册、兑换与 API 密钥配置教程",
    ),
    GuideSpec(
        "codex-guide.md",
        f"{OUTPUT_PARENT_DIR}/01-Codex-API-登录对接教程.md",
        "## 第三章 Codex 客户端接入",
        "Codex API 登录对接教程",
        lead_override=CHILD_GUIDE_LEAD,
        tutorial_points_override=(
            "下载并初始化 Codex",
            "配置 `config.toml` 和 `auth.json`",
            "使用 API Key 登录",
            "验证配置并排查常见错误",
        ),
        preface="""## 开始前准备

在父教程中创建密钥后，点击“使用密钥”，打开 Codex 配置区域。优先复制弹窗中已填好的 `base_url` 和 `api_key`，不要复制教程截图中的脱敏密钥。

![使用 API 密钥弹窗中的 Codex 配置示例，密钥已脱敏](../../frontend/public/img/codex-guide/image-5.png)

图：Codex 配置弹窗示例。请使用自己页面中生成的 API Key。
""",
        body_end_heading="## 第五章 FAQ",
        removed_sections=((
            "### 3.2 其他客户端独立教程",
            "### 3.3 重新打开 Codex 并使用 API 登录",
        ),),
        replacements=(
            ("## 第三章 Codex 客户端接入", "## Codex 客户端配置流程"),
            ("### 3.1 手动配置 Codex 系列", "### 1. 手动配置 Codex 系列"),
            ("### 3.3 重新打开 Codex 并使用 API 登录", "### 2. 重新打开 Codex 并使用 API 登录"),
            ("## 第四章 验证与排错", "## 验证与排错"),
            ("### 4.1 一行命令自检（推荐）", "### 1. 一行命令自检（推荐）"),
            ("### 4.2 登录失败时快速检查", "### 2. 登录失败时快速检查"),
            ("### 4.3 常见报错对照表", "### 3. 常见报错对照表"),
            ("图 8：", "图 1："),
            ("图 9：", "图 2："),
            ("图 10：", "图 3："),
            ("图 11：", "图 4："),
            ("图 12：", "图 5："),
            ("图 13：", "图 6："),
            ("图 14：", "图 7："),
            ("图 15：", "图 8："),
            ("图 16：", "图 9："),
            ("配置前仍需先完成第二章创建 API Key。", "配置前请先完成父教程。"),
            ("按第二章“使用密钥”弹窗中的", "按父教程“使用密钥”弹窗中的"),
            ("回第二章重新创建或复制密钥", "回到父教程重新创建或复制密钥"),
        ),
    ),
    GuideSpec(
        "claude-code-guide.md",
        f"{OUTPUT_PARENT_DIR}/02-Claude-Code-配置教程.md",
        "### 2. 手动配置 Claude Code",
        "Claude Code 配置教程",
        lead_override=CHILD_GUIDE_LEAD,
        tutorial_points_override=(
            "定位 Claude 配置目录",
            "写入 `settings.json`",
            "可选的系统环境变量配置",
            "启动 Claude Code 验证",
        ),
        preface="""## 开始前准备

在父教程中点击“使用密钥”，切换到 Claude Code 配置区域，复制弹窗里的真实 `base_url` 和 `api_key`。

![Claude Code 配置弹窗示例，密钥已脱敏](../../frontend/public/img/codex-guide/image-22.png)

图：Claude Code 配置示例。截图中的 API Key 已脱敏，请以自己的弹窗为准。
""",
        replacements=(
            ("### 2. 手动配置 Claude Code", "## Claude Code 配置流程"),
            ("#### 2.1 定位 Claude 配置目录", "### 1. 定位 Claude 配置目录"),
            ("#### 2.2 方式 A：写入 `settings.json`（推荐）", "### 2. 方式 A：写入 `settings.json`（推荐）"),
            ("#### 2.3 方式 B：配置系统环境变量", "### 3. 方式 B：配置系统环境变量"),
            ("### 3. 验证与排错", "## 验证与排错"),
        ),
    ),
    GuideSpec(
        "open-code-guide.md",
        f"{OUTPUT_PARENT_DIR}/03-Open-Code-配置教程.md",
        "### 2. 安装并首次启动 Open Code",
        "Open Code 配置教程",
        lead_override=CHILD_GUIDE_LEAD,
        tutorial_points_override=(
            "安装并首次启动 Open Code",
            "配置 `opencode.json`",
            "使用 `/connect` 临时切换",
            "验证 provider 和模型",
        ),
        replacements=(
            ("### 2. 安装并首次启动 Open Code", "## Open Code 配置流程\n\n### 1. 安装并首次启动 Open Code"),
            ("### 3. 定位 Open Code 配置目录", "### 2. 定位 Open Code 配置目录"),
            ("### 4. 方式 A：写入 `opencode.json`（推荐，长期生效）", "### 3. 方式 A：写入 `opencode.json`（推荐，长期生效）"),
            ("### 5. 方式 B：客户端内 `/connect` 临时切换", "### 4. 方式 B：客户端内 `/connect` 临时切换"),
            ("### 6. 验证与排错", "## 验证与排错"),
        ),
    ),
    GuideSpec(
        "open-claw-guide.md",
        f"{OUTPUT_PARENT_DIR}/04-Open-Claw-配置教程.md",
        "### 2. 方式 A：腾讯云在线配置（推荐新手）",
        "Open Claw 配置教程",
        lead_override=CHILD_GUIDE_LEAD,
        tutorial_points_override=(
            "腾讯云在线配置",
            "Windows / macOS / Linux 本地配置",
            "区分 `openai-completions` 与 `openai-responses`",
            "验证地址、密钥和模型",
        ),
        replacements=(
            ("### 2. 方式 A：腾讯云在线配置（推荐新手）", "## Open Claw 配置流程\n\n### 1. 方式 A：腾讯云在线配置（推荐新手）"),
            ("### 3. 方式 B：本地配置（Windows / macOS / Linux）", "### 2. 方式 B：本地配置（Windows / macOS / Linux）"),
            ("### 4. 验证与快速检查", "## 验证与快速检查"),
        ),
    ),
    GuideSpec(
        "mobile-guide.md",
        f"{OUTPUT_PARENT_DIR}/05-移动端-Chatbox-配置教程.md",
        "### 1. 前往官网下载 Chatbox",
        "移动端 Chatbox 配置教程",
        lead_override=CHILD_GUIDE_LEAD,
        tutorial_points_override=(
            "下载并安装 Chatbox",
            "添加 OpenAI response API 兼容提供方",
            "获取并选择模型",
            "新建对话完成验证",
        ),
        replacements=((
            "### 1. 前往官网下载 Chatbox",
            "## Chatbox 配置流程\n\n### 1. 前往官网下载 Chatbox",
        ),),
    ),
    GuideSpec(
        "image-guide.md",
        f"{OUTPUT_PARENT_DIR}/06-Cherry-Studio-图像生成教程.md",
        "### 当前图像生成路径说明",
        "Cherry Studio 图像生成教程",
        lead_override=CHILD_GUIDE_LEAD,
        tutorial_points_override=(
            "安装 Cherry Studio",
            "配置 New API 模型服务",
            "添加 `gpt-image-2` 图像生成模型",
            "使用绘画入口生成图片",
        ),
        replacements=((
            "### 当前图像生成路径说明",
            "## Cherry Studio 图像生成流程\n\n### 当前图像生成路径说明",
        ),),
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
    lead = guide.lead_override or extract_section(
        source,
        "引导文案：\n\n",
        "\n\n教程要点：",
    )
    if guide.tutorial_points_override is None:
        tutorial_points = extract_section(
            source,
            "教程要点：\n\n",
            "\n\n章节快捷入口：",
        )
    else:
        tutorial_points = "\n".join(
            f"- {point}" for point in guide.tutorial_points_override
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


def remove_sections(
    markdown: str,
    section_ranges: tuple[tuple[str, str], ...],
) -> str:
    rewritten = markdown
    for start_heading, end_heading in section_ranges:
        start_offset = rewritten.find(start_heading)
        end_offset = rewritten.find(end_heading, start_offset + len(start_heading))
        if start_offset < 0 or end_offset < 0:
            raise ValueError(f"找不到待删除区段: {start_heading}")
        rewritten = (
            f"{rewritten[:start_offset].rstrip()}\n\n"
            f"{rewritten[end_offset:]}"
        )
    return rewritten


def remove_figure(markdown: str, image_line: str, caption: str) -> str:
    figure = f"{image_line}\n\n{caption}\n\n"
    if figure not in markdown:
        raise ValueError(f"找不到待移除的教程截图: {caption}")
    return markdown.replace(figure, "")


def sanitize_upload_markdown(markdown: str, guide: GuideSpec) -> str:
    if guide.source_name != "image-guide.md":
        return markdown
    safe_markdown = remove_figure(
        markdown,
        "![Cherry Studio 填写 API 地址和密钥]"
        "(../../frontend/public/img/image-guide/image-6.png)",
        "图 4：填写 API 地址和自己的 API 密钥。",
    )
    safe_markdown = remove_figure(
        safe_markdown,
        "![Cherry Studio 获取模型列表]"
        "(../../frontend/public/img/image-guide/image-7.png)",
        "图 5：点击获取模型列表。",
    )
    safe_markdown = remove_figure(
        safe_markdown,
        "![Cherry Studio 图像生成模型配置完成]"
        "(../../frontend/public/img/image-guide/image-10.png)",
        "图 8：配置完成后的模型服务状态。",
    )
    safe_markdown = safe_markdown.replace(
        "../../frontend/public/img/image-guide/image-11.png",
        "../../frontend/public/img/image-guide/image-12.png",
    ).replace(
        "图 9：点击上方加号。",
        "图 9：点击上方加号后，选择“绘画”入口。",
    )
    return safe_markdown


def render_guide_markdown(guide: GuideSpec) -> str:
    source_path = SOURCE_DIR / validated_markdown_name(guide.source_name)
    source = source_path.read_text(encoding="utf-8")
    heading_offset = source.find(guide.body_heading)
    if heading_offset < 0:
        raise ValueError(f"找不到教程正文起始标题: {guide.body_heading}")
    body = source[heading_offset:]
    if guide.body_end_heading is not None:
        end_offset = body.find(guide.body_end_heading)
        if end_offset < 0:
            raise ValueError(f"找不到教程正文结束标题: {guide.body_end_heading}")
        body = body[:end_offset].rstrip()
    body = remove_sections(body, guide.removed_sections)
    for old_text, new_text in guide.replacements:
        if old_text not in body:
            raise ValueError(f"找不到待改写内容: {old_text}")
        body = body.replace(old_text, new_text)
    introduction = render_introduction(source, guide)
    rendered_markdown = rewrite_for_feishu(
        f"{introduction}{guide.preface}{body.rstrip()}\n"
    )
    return sanitize_upload_markdown(rendered_markdown, guide)


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


def export_guides(output_dir: Path, exports: tuple[tuple[str, str], ...]) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    for output_name, content in exports:
        destination = output_dir / validated_markdown_path(output_name)
        destination.parent.mkdir(parents=True, exist_ok=True)
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
