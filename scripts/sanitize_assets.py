#!/usr/bin/env python3
"""Anonymize marketing screenshots before publishing to assets/."""

from __future__ import annotations

import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
SRC = Path(os.environ.get("SANITIZE_SRC", ""))
OUT = ROOT / "assets"

NAME_COLOR = (248, 250, 252)
ID_COLOR = (100, 116, 139)
IMAGE_COLOR = (226, 232, 240)
TAG_COLOR = (100, 116, 139)

DEMO_FLEET = [
    ("docklog", "docklog-app", "latest", "a1b2c3d4e5f6"),
    ("web-app-1", "web-app", "latest", "b2c3d4e5f6a1"),
    ("postgres-db", "postgres", "16-alpine", "c3d4e5f6a1b2"),
    ("redis-cache", "redis", "7-alpine", "d4e5f6a1b2c3"),
    ("loadtest-app", "loadtest-demo", "latest", "e5f6a1b2c3d4"),
    ("worker-queue", "api-backend", "latest", "f6a1b2c3d4e5"),
    ("api-backend", "api-backend", "latest", "a7b8c9d0e1f2"),
    ("frontend-web", "frontend-web", "latest", "b8c9d0e1f2a3"),
]

DEMO_STOPPED = [
    ("web-app-1", "web-app", "latest", "b2c3d4e5f6a1"),
    ("redis-cache", "redis", "7-alpine", "d4e5f6a1b2c3"),
]


def load_font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    candidates = [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf" if bold else "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/System/Library/Fonts/Helvetica.ttc",
        "/System/Library/Fonts/Menlo.ttc",
    ]
    for path in candidates:
        if os.path.exists(path):
            try:
                return ImageFont.truetype(path, size)
            except OSError:
                continue
    return ImageFont.load_default()


NAME_FONT = load_font(13, bold=True)
ID_FONT = load_font(10)
IMAGE_FONT = load_font(12)
TAG_FONT = load_font(9)
AUDIT_RESOURCE_FONT = load_font(12, bold=True)
AUDIT_DETAIL_FONT = load_font(10)

RESOURCE_COLOR = (8, 145, 178)
DETAIL_COLOR = (148, 163, 184)

# Measured row positions in 1024px-wide screenshots
CONTAINERS_ROW_TOPS = [222, 263, 305, 347, 388, 430, 471, 513]
DASHBOARD_ROW_TOPS = [359, 401]
AUDIT_ROW_TOPS = [294, 327, 360, 393, 427, 460, 493, 526]
ROW_HEIGHT = 41
DASHBOARD_ROW_HEIGHT = 42
AUDIT_ROW_HEIGHT = 33

AUDIT_RESOURCE = "docklog"
AUDIT_DETAILS = [
    "Interactive shell session closed",
    "Executed command: cd app",
    "Executed command: ls",
    "Executed command: cd src",
    "Executed command: pwd",
    "Executed command: ls",
    "Executed command: last",
    "Executed command: ls -la",
]


def sample_row_bg(im: Image.Image, y: int) -> tuple[int, int, int]:
    return im.getpixel((520, min(y + 20, im.size[1] - 1)))


def write_container_row(
    draw: ImageDraw.ImageDraw,
    im: Image.Image,
    y: int,
    name: str,
    image_repo: str,
    image_tag: str,
    fake_id: str,
    *,
    row_height: int = 42,
) -> None:
    bg = sample_row_bg(im, y)
    draw.rectangle((210, y, 620, y + row_height), fill=bg)
    draw.text((272, y + 8), name, fill=NAME_COLOR, font=NAME_FONT)
    draw.text((272, y + 26), fake_id, fill=ID_COLOR, font=ID_FONT)
    draw.text((478, y + 10), image_repo, fill=IMAGE_COLOR, font=IMAGE_FONT)
    draw.text((478, y + 26), image_tag, fill=TAG_COLOR, font=TAG_FONT)


def save_asset(img: Image.Image, dest: Path) -> None:
    if dest.suffix.lower() == ".webp":
        img.save(dest, format="WEBP", quality=88, method=6)
    else:
        img.save(dest, optimize=True)


def sanitize_containers(src: Path, dest: Path) -> None:
    img = Image.open(src).convert("RGB")
    draw = ImageDraw.Draw(img)
    for i, row in enumerate(CONTAINERS_ROW_TOPS):
        if i >= len(DEMO_FLEET):
            break
        write_container_row(draw, img, row - 2, *DEMO_FLEET[i])
    save_asset(img, dest)


def sanitize_dashboard(src: Path, dest: Path) -> None:
    img = Image.open(src).convert("RGB")
    draw = ImageDraw.Draw(img)
    for i, row in enumerate(DASHBOARD_ROW_TOPS):
        if i >= len(DEMO_STOPPED):
            break
        write_container_row(draw, img, row - 2, *DEMO_STOPPED[i], row_height=DASHBOARD_ROW_HEIGHT)
    save_asset(img, dest)


def sanitize_audit_logs(src: Path, dest: Path) -> None:
    img = Image.open(src).convert("RGB")
    draw = ImageDraw.Draw(img)

    for i, row in enumerate(AUDIT_ROW_TOPS):
        bg = sample_row_bg(img, row)
        draw.rectangle((648, row, 718, row + AUDIT_ROW_HEIGHT), fill=bg)
        draw.text((658, row + 8), AUDIT_RESOURCE, fill=RESOURCE_COLOR, font=AUDIT_RESOURCE_FONT)

        if i >= len(AUDIT_DETAILS):
            continue
        draw.rectangle((718, row, 995, row + AUDIT_ROW_HEIGHT), fill=bg)
        draw.text((728, row + 9), AUDIT_DETAILS[i], fill=DETAIL_COLOR, font=AUDIT_DETAIL_FONT)

    save_asset(img, dest)


def copy_as_is(src: Path, dest: Path) -> None:
    save_asset(Image.open(src).convert("RGB"), dest)


def main() -> None:
    if not SRC.exists():
        raise SystemExit(f"Source directory not found: {SRC}")

    mappings = {
        "image-bca22829-de8b-443c-9a97-1bd3859ab6b0.png": ("dashboard.webp", sanitize_dashboard),
        "image-72dfcd95-c527-4760-9104-015e7f55a7b8.png": ("containers.webp", sanitize_containers),
        "image-b631dba8-2271-4bee-80f9-607e8c6b3510.png": ("rbac.webp", copy_as_is),
        "image-a2788c14-d836-4f84-9b17-10cdad7436e0.png": ("health.webp", copy_as_is),
        "image-c27c3e27-46bc-43f8-b9bf-c7a8b65407b0.png": ("audit-logs.webp", sanitize_audit_logs),
    }

    OUT.mkdir(parents=True, exist_ok=True)

    for src_name, (dest_name, handler) in mappings.items():
        src_path = SRC / src_name
        if not src_path.exists():
            print(f"skip missing: {src_name}")
            continue
        dest_path = OUT / dest_name
        handler(src_path, dest_path)
        print(f"wrote {dest_path.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
