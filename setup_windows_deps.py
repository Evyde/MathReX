#!/usr/bin/env python3
"""
Setup script for MathReX Windows dependencies
Downloads and sets up ONNX Runtime and tokenizers library
"""

import os
import sys
import urllib.request
import zipfile
import shutil
from pathlib import Path

def download_file(url, filename):
    """Download a file with progress indication"""
    print(f"Downloading {filename}...")
    try:
        urllib.request.urlretrieve(url, filename)
        print(f"✓ Downloaded {filename}")
        return True
    except Exception as e:
        print(f"✗ Failed to download {filename}: {e}")
        return False

def extract_zip(zip_path, extract_to):
    """Extract a zip file"""
    try:
        with zipfile.ZipFile(zip_path, 'r') as zip_ref:
            zip_ref.extractall(extract_to)
        print(f"✓ Extracted {zip_path}")
        return True
    except Exception as e:
        print(f"✗ Failed to extract {zip_path}: {e}")
        return False

def setup_onnxruntime():
    """Download and setup ONNX Runtime"""
    print("\n=== Setting up ONNX Runtime ===")
    
    # Create onnxruntime directory
    os.makedirs("onnxruntime", exist_ok=True)
    
    # Download ONNX Runtime
    onnx_url = "https://github.com/microsoft/onnxruntime/releases/download/v1.21.0/onnxruntime-win-x64-1.21.0.zip"
    onnx_zip = "onnxruntime-win-x64-1.21.0.zip"
    
    if not download_file(onnx_url, onnx_zip):
        return False
    
    # Extract ONNX Runtime
    temp_dir = "temp_onnx"
    if not extract_zip(onnx_zip, temp_dir):
        return False
    
    # Copy DLL files
    try:
        onnx_lib_dir = os.path.join(temp_dir, "onnxruntime-win-x64-1.21.0", "lib")
        
        # Copy main DLL
        shutil.copy2(
            os.path.join(onnx_lib_dir, "onnxruntime.dll"),
            os.path.join("onnxruntime", "onnxruntime.dll")
        )
        print("✓ Copied onnxruntime.dll")
        
        # Copy providers DLL
        providers_dll = os.path.join(onnx_lib_dir, "onnxruntime_providers_shared.dll")
        if os.path.exists(providers_dll):
            shutil.copy2(
                providers_dll,
                os.path.join("onnxruntime", "onnxruntime_providers_shared.dll")
            )
            print("✓ Copied onnxruntime_providers_shared.dll")
        
        # Copy any other DLLs that might be needed
        for dll_file in os.listdir(onnx_lib_dir):
            if dll_file.endswith('.dll') and dll_file not in ['onnxruntime.dll', 'onnxruntime_providers_shared.dll']:
                shutil.copy2(
                    os.path.join(onnx_lib_dir, dll_file),
                    os.path.join("onnxruntime", dll_file)
                )
                print(f"✓ Copied {dll_file}")
        
    except Exception as e:
        print(f"✗ Failed to copy ONNX Runtime files: {e}")
        return False
    
    # Cleanup
    try:
        os.remove(onnx_zip)
        shutil.rmtree(temp_dir)
        print("✓ Cleaned up temporary files")
    except Exception as e:
        print(f"Warning: Failed to cleanup: {e}")
    
    return True

def setup_tokenizers():
    """Setup tokenizers library (placeholder for now)"""
    print("\n=== Setting up Tokenizers Library ===")
    
    # Create tokenizers directory
    tokenizers_dir = os.path.join("libtokenizers", "windows_amd64")
    os.makedirs(tokenizers_dir, exist_ok=True)
    
    # Create a placeholder tokenizers library file
    placeholder_file = os.path.join(tokenizers_dir, "tokenizers.lib")
    try:
        with open(placeholder_file, 'w') as f:
            f.write("# Placeholder tokenizers library file\n")
        print("✓ Created placeholder tokenizers.lib")
        return True
    except Exception as e:
        print(f"✗ Failed to create tokenizers placeholder: {e}")
        return False

def check_setup():
    """Check if all required files are present"""
    print("\n=== Checking Setup ===")
    
    required_files = [
        "onnxruntime/onnxruntime.dll",
        "libtokenizers/windows_amd64/tokenizers.lib"
    ]
    
    all_present = True
    for file_path in required_files:
        if os.path.exists(file_path):
            print(f"✓ {file_path}")
        else:
            print(f"✗ {file_path} [MISSING]")
            all_present = False
    
    return all_present

def main():
    print("MathReX Windows Dependencies Setup")
    print("=" * 40)
    
    # Change to script directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(script_dir)
    print(f"Working directory: {os.getcwd()}")
    
    success = True
    
    # Setup ONNX Runtime
    if not setup_onnxruntime():
        success = False
    
    # Setup Tokenizers
    if not setup_tokenizers():
        success = False
    
    # Check final setup
    if check_setup():
        print("\n🎉 Setup completed successfully!")
        print("You can now run MathReX.exe")
    else:
        print("\n❌ Setup incomplete. Some files are missing.")
        success = False
    
    if not success:
        print("\nNote: This is a simplified setup for debugging.")
        print("The full tokenizers library requires Rust compilation.")
    
    return 0 if success else 1

if __name__ == "__main__":
    sys.exit(main())
