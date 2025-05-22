import os
import requests
import shutil
import tarfile
import zipfile
import platform
import argparse
from pathlib import Path

# 配置 onnxruntime 版本和目标平台
# 您可以根据需要更改此版本
ONNXRUNTIME_VERSION = "1.21.0" # Updated version based on user feedback

# 目标平台配置
# (GitHub OS, GitHub Arch, Go OS, Go Arch, Archive Extension, Archive Filename Pattern on GitHub)
TARGET_PLATFORMS = [
    ("win", "x64", "windows", "amd64", "zip", "onnxruntime-win-x64-{version}.zip"),
    ("osx", "x86_64", "darwin", "amd64", "tgz", "onnxruntime-osx-x86_64-{version}.tgz"), # Changed "x64" to "x86_64" for GitHub arch
    ("osx", "arm64", "darwin", "arm64", "tgz", "onnxruntime-osx-arm64-{version}.tgz"),
    # 可以根据需要添加更多平台，例如 Linux
    ("linux", "x64", "linux", "amd64", "tgz", "onnxruntime-linux-x64-{version}.tgz"),
    ("linux", "aarch64", "linux", "arm64", "tgz", "onnxruntime-linux-aarch64-{version}.tgz"),
    ("win", "arm64", "windows", "arm64", "zip", "onnxruntime-win-arm64-{version}.zip"),
]

# 项目中 onnxruntime 库的基础路径
LIB_BASE_PATH = Path(__file__).parent / "onnxruntime"
TEMP_DOWNLOAD_DIR = Path(__file__).parent / "onnxruntime" / "temp_onnx_download"

def download_file(url, dest_path):
    """下载文件到指定路径"""
    print(f"Downloading {url} to {dest_path}...")
    try:
        response = requests.get(url, stream=True, timeout=300) # 5分钟超时
        response.raise_for_status()  # 如果请求失败则引发 HTTPError
        with open(dest_path, "wb") as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
        print("Download complete.")
        return True
    except requests.exceptions.RequestException as e:
        print(f"Error downloading {url}: {e}")
        return False

def extract_archive(archive_path, extract_to_dir):
    """解压归档文件"""
    print(f"Extracting {archive_path} to {extract_to_dir}...")
    try:
        if archive_path.name.endswith(".zip"):
            with zipfile.ZipFile(archive_path, "r") as zip_ref:
                zip_ref.extractall(extract_to_dir)
        elif archive_path.name.endswith(".tgz") or archive_path.name.endswith(".tar.gz"):
            with tarfile.open(archive_path, "r:gz") as tar_ref:
                tar_ref.extractall(extract_to_dir)
        else:
            print(f"Unsupported archive format: {archive_path.name}")
            return False
        print("Extraction complete.")
        return True
    except Exception as e:
        print(f"Error extracting {archive_path}: {e}")
        return False

def organize_files(extracted_dir_path, target_lib_path, version, gh_os, gh_arch):
    """
    将解压后的文件组织到目标目录结构。
    onnxruntime 的目录结构通常是：
    extracted_dir_path / onnxruntime-<os>-<arch>-<version> / include / ...
    extracted_dir_path / onnxruntime-<os>-<arch>-<version> / lib / onnxruntime.dll (or .so, .dylib)
    """
    print(f"Organizing files from {extracted_dir_path} to {target_lib_path}...")
    # 构造实际包含内容的子目录名
    # 例如 onnxruntime-win-x64-1.17.3
    content_subdir_name_pattern = f"onnxruntime-{gh_os}-{gh_arch}-{version}"
    
    # 查找实际的解压出来的文件夹名 (有些归档可能不完全遵循模式)
    actual_content_subdir = None
    for item in extracted_dir_path.iterdir():
        if item.is_dir() and item.name.startswith(f"onnxruntime-{gh_os}-{gh_arch}"):
            actual_content_subdir = item
            break
    
    if not actual_content_subdir:
        print(f"Error: Could not find content subdirectory matching pattern '{content_subdir_name_pattern}' in {extracted_dir_path}")
        # 有些归档可能直接解压内容到 extract_to_dir，检查这种情况
        if (extracted_dir_path / "include").exists() and (extracted_dir_path / "lib").exists():
            actual_content_subdir = extracted_dir_path
            print("Found include/lib directly in extraction path. Proceeding.")
        else:
            return False

    source_include_dir = actual_content_subdir / "include"
    source_lib_dir = actual_content_subdir / "lib"

    if not source_include_dir.exists() or not source_lib_dir.exists():
        print(f"Error: 'include' or 'lib' directory not found in {actual_content_subdir}")
        return False

    # 创建目标目录
    target_lib_path.mkdir(parents=True, exist_ok=True)
    target_include_dir = target_lib_path / "include"

    # 清理旧文件（如果存在）
    if target_include_dir.exists():
        shutil.rmtree(target_include_dir)
    # 清理目标库目录中可能的旧库文件
    for item in target_lib_path.glob("onnxruntime.*"):
        item.unlink()
    for item in target_lib_path.glob("libonnxruntime.*"):
        item.unlink()


    # 复制 include 文件夹
    shutil.copytree(source_include_dir, target_include_dir)
    print(f"Copied include files to {target_include_dir}")

    # 复制 lib 文件 (dll, so, dylib)
    # 注意：Go CGO 通常希望库文件直接在指定的库路径下，而不是在子目录 lib/ 中
    # 所以我们将 lib/ 目录下的文件复制到 target_lib_path
    copied_libs = []
    for item in source_lib_dir.iterdir():
        if item.is_file() and (item.name.startswith("onnxruntime.") or item.name.startswith("libonnxruntime.")):
            shutil.copy2(item, target_lib_path / item.name)
            copied_libs.append(item.name)
    
    if not copied_libs:
        print(f"Warning: No library files (onnxruntime.*, libonnxruntime.*) found in {source_lib_dir}")
    else:
        print(f"Copied library files to {target_lib_path}: {', '.join(copied_libs)}")

    print("File organization complete.")
    return True

def main():
    parser = argparse.ArgumentParser(description=f"Download and set up ONNX Runtime v{ONNXRUNTIME_VERSION}.")
    parser.add_argument(
        "--platforms",
        nargs="+",
        help="Specify platforms to download, e.g., windows-amd64 osx-arm64. Default is all configured platforms.",
        default=None
    )
    parser.add_argument(
        "--version",
        type=str,
        default=ONNXRUNTIME_VERSION,
        help=f"Specify ONNX Runtime version to download. Default is {ONNXRUNTIME_VERSION}."
    )

    args = parser.parse_args()
    current_version = args.version

    # 确定要处理的平台
    platforms_to_process = []
    if args.platforms:
        for p_str in args.platforms:
            try:
                p_parts = p_str.split('-')
                req_go_os = p_parts[0]
                req_go_arch = p_parts[1]
                found = False
                for gh_os_cfg, gh_arch_cfg, go_os_cfg, go_arch_cfg, _, archive_filename_pattern_cfg in TARGET_PLATFORMS:
                    if go_os_cfg == req_go_os and go_arch_cfg == req_go_arch:
                        # Use the archive_filename_pattern directly
                        archive_name = archive_filename_pattern_cfg.format(version=current_version)
                        platforms_to_process.append((gh_os_cfg, gh_arch_cfg, go_os_cfg, go_arch_cfg, archive_name))
                        found = True
                        break
                if not found:
                    print(f"Warning: Platform string '{p_str}' (GoOS: {req_go_os}, GoArch: {req_go_arch}) not recognized or configured. Skipping.")
            except IndexError:
                print(f"Warning: Invalid platform string format '{p_str}'. Should be goos-goarch (e.g., windows-amd64). Skipping.")
    else: #默认处理所有配置的平台
        platforms_to_process = [(gh_os, gh_arch, go_os, go_arch, archive_filename_pattern.format(version=current_version))
                                for gh_os, gh_arch, go_os, go_arch, _, archive_filename_pattern in TARGET_PLATFORMS]


    if not TEMP_DOWNLOAD_DIR.exists():
        TEMP_DOWNLOAD_DIR.mkdir(parents=True)

    success_count = 0
    # gh_os, gh_arch, go_os, go_arch, archive_name (already formatted with version)
    for gh_os_val, gh_arch_val, go_os_val, go_arch_val, archive_name_val in platforms_to_process:
        print(f"\nProcessing: {go_os_val}-{go_arch_val} (ONNX Runtime GitHub: {gh_os_val}-{gh_arch_val})")
        
        # archive_name_val is now the full filename like "onnxruntime-win-x64-1.21.0.zip"
        download_url = f"https://github.com/microsoft/onnxruntime/releases/download/v{current_version}/{archive_name_val}"
        
        downloaded_archive_path = TEMP_DOWNLOAD_DIR / archive_name_val
        
        # 目标库路径，例如 backend/Go/lib/onnxruntime/amd64_windows/
        target_lib_path = LIB_BASE_PATH / f"{go_arch_val}_{go_os_val}"

        # 1. 下载
        if not download_file(download_url, downloaded_archive_path):
            print(f"Failed to download for {go_os}-{go_arch}. Skipping.")
            continue

        # 2. 解压
        # 解压到一个临时子目录，以避免不同归档之间的文件名冲突
        temp_extract_specific_dir = TEMP_DOWNLOAD_DIR / f"extracted_{gh_os_val}_{gh_arch_val}"
        if temp_extract_specific_dir.exists():
            shutil.rmtree(temp_extract_specific_dir) # 清理旧的解压内容
        temp_extract_specific_dir.mkdir()

        if not extract_archive(downloaded_archive_path, temp_extract_specific_dir):
            print(f"Failed to extract for {go_os_val}-{go_arch_val}. Skipping.")
            continue
        
        # 3. 组织文件
        # The extracted directory name pattern is implicitly handled by organize_files
        # which looks for a subdir starting with onnxruntime-<gh_os>-<gh_arch>
        if not organize_files(temp_extract_specific_dir, target_lib_path, current_version, gh_os_val, gh_arch_val):
            print(f"Failed to organize files for {go_os_val}-{go_arch_val}. Skipping.")
            continue
        
        print(f"Successfully set up ONNX Runtime for {go_os_val}-{go_arch_val} at {target_lib_path}")
        success_count += 1

    # 清理临时下载目录
    if TEMP_DOWNLOAD_DIR.exists():
        try:
            shutil.rmtree(TEMP_DOWNLOAD_DIR)
            print(f"\nCleaned up temporary directory: {TEMP_DOWNLOAD_DIR}")
        except Exception as e:
            print(f"Warning: Could not clean up temporary directory {TEMP_DOWNLOAD_DIR}: {e}")

    print(f"\nSetup complete. Successfully processed {success_count}/{len(platforms_to_process)} requested platform(s).")

if __name__ == "__main__":
    main()
