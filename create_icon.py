#!/usr/bin/env python3
"""
Generate MathReX application icon
"""
from PIL import Image, ImageDraw, ImageFont
import os

def create_icon():
    # Create a 512x512 image with transparent background
    size = 512
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    
    # Background circle with gradient-like effect
    center = size // 2
    radius = size // 2 - 20
    
    # Draw background circle
    draw.ellipse([center - radius, center - radius, center + radius, center + radius], 
                fill=(70, 130, 180, 255), outline=(30, 90, 140, 255), width=4)
    
    # Draw mathematical symbols
    try:
        # Try to use a system font
        font_size = size // 6
        font = ImageFont.truetype("/System/Library/Fonts/Arial.ttf", font_size)
    except:
        try:
            font = ImageFont.truetype("arial.ttf", font_size)
        except:
            font = ImageFont.load_default()
    
    # Mathematical formula: ∫ x² dx
    formula = "∫x²dx"
    
    # Get text bounding box
    bbox = draw.textbbox((0, 0), formula, font=font)
    text_width = bbox[2] - bbox[0]
    text_height = bbox[3] - bbox[1]
    
    # Center the text
    x = (size - text_width) // 2
    y = (size - text_height) // 2 - 20
    
    # Draw text with shadow
    draw.text((x + 2, y + 2), formula, fill=(0, 0, 0, 128), font=font)
    draw.text((x, y), formula, fill=(255, 255, 255, 255), font=font)
    
    # Add "ReX" text below
    try:
        rex_font = ImageFont.truetype("/System/Library/Fonts/Arial.ttf", font_size // 2)
    except:
        try:
            rex_font = ImageFont.truetype("arial.ttf", font_size // 2)
        except:
            rex_font = ImageFont.load_default()
    
    rex_text = "ReX"
    rex_bbox = draw.textbbox((0, 0), rex_text, font=rex_font)
    rex_width = rex_bbox[2] - rex_bbox[0]
    rex_x = (size - rex_width) // 2
    rex_y = y + text_height + 10
    
    draw.text((rex_x + 1, rex_y + 1), rex_text, fill=(0, 0, 0, 128), font=rex_font)
    draw.text((rex_x, rex_y), rex_text, fill=(255, 255, 255, 255), font=rex_font)
    
    return img

def main():
    print("Creating MathReX icon...")
    
    # Create the icon
    icon = create_icon()
    
    # Save as PNG
    icon.save("icon.png", "PNG")
    print("Saved icon.png")
    
    # Create ICO for Windows
    icon.save("icon.ico", "ICO", sizes=[(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
    print("Saved icon.ico")
    
    # Create ICNS for macOS (requires pillow-heif or manual conversion)
    try:
        # Create different sizes for ICNS
        sizes = [16, 32, 64, 128, 256, 512]
        icons = []
        for size in sizes:
            resized = icon.resize((size, size), Image.Resampling.LANCZOS)
            icons.append(resized)
        
        # Save the largest as a temporary file for manual ICNS creation
        icon.save("icon_512.png", "PNG")
        print("Saved icon_512.png (use this to create ICNS manually)")
        
    except Exception as e:
        print(f"Could not create ICNS: {e}")
    
    print("Icon creation completed!")

if __name__ == "__main__":
    main()
