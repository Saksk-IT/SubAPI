"""V1 Word 导出的安全路径、原子文件写入与检查。"""

from __future__ import annotations

import os
import tempfile
from pathlib import Path


def validated_docx_path(relative_path: str) -> Path:
    path = Path(relative_path)
    if (
        path.is_absolute()
        or path.suffix.lower() != ".docx"
        or not path.parts
        or any(part in {"", ".", ".."} for part in path.parts)
    ):
        raise ValueError(f"Word 输出路径无效: {relative_path}")
    return path


def atomic_write_bytes(destination: Path, content: bytes) -> None:
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="wb",
            dir=destination.parent,
            prefix=f".{destination.name}.",
            suffix=".tmp",
            delete=False,
        ) as temporary_file:
            temporary_path = Path(temporary_file.name)
            temporary_file.write(content)
            temporary_file.flush()
            os.fsync(temporary_file.fileno())
        temporary_path.chmod(0o644)
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
        raise ValueError(f"Word 输出目录超出允许范围: {relative_path}") from error
    return resolved_parent / destination.name


def export_documents(
    output_dir: Path,
    exports: tuple[tuple[str, bytes], ...],
) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    for output_name, content in exports:
        destination = safe_output_destination(
            output_dir,
            validated_docx_path(output_name),
        )
        atomic_write_bytes(destination, content)


def check_documents(
    output_dir: Path,
    exports: tuple[tuple[str, bytes], ...],
) -> list[str]:
    errors: list[str] = []
    expected_names = {output_name for output_name, _ in exports}
    existing_names = {
        path.relative_to(output_dir).as_posix()
        for path in output_dir.rglob("*.docx")
    }
    for missing_name in sorted(expected_names - existing_names):
        errors.append(f"缺少生成文件: {missing_name}")
    for extra_name in sorted(existing_names - expected_names):
        errors.append(f"存在多余生成文件: {extra_name}")
    for output_name, expected_content in exports:
        output_path = output_dir / validated_docx_path(output_name)
        if output_path.is_symlink() or (
            output_path.exists() and not output_path.is_file()
        ):
            errors.append(f"生成路径不是普通文件: {output_name}")
            continue
        if output_path.is_file() and output_path.read_bytes() != expected_content:
            errors.append(f"生成文件已过期: {output_name}")
    return errors
