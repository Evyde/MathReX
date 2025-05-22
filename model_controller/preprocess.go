package model_controller

import (
	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
	"image"
	"image/draw"
	"io"
)

// PreprocessToModelFormat 输入图像路径，返回符合 TrOCR 模型的张量（[1,3,384,384]）和形状信息
// 输出格式：数据为 []float32（CHW 顺序），形状为 []int64{1, 3, 384, 384}
func PreprocessToModelFormat(file io.Reader) ([]float32, []int64, error) {
	img, err := imaging.Decode(file)

	if err != nil {
		return nil, nil, err
	}

	// 2. 转换为 RGBA 格式（兼容所有输入类型）
	rgba := toRGBA(img)

	// 3. Resize 到 384x384（使用 Bicubic 插值，对应配置的 resample=3）
	targetW, targetH := 384, 384
	resized := resize.Resize(uint(targetW), uint(targetH), rgba, resize.Bicubic) // 重要：对齐配置的 resample=3

	// 4. 预处理：Rescale → Normalize
	tensor := make([]float32, 3*targetH*targetW) // CHW 顺序 [3, 384, 384]

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			// 获取像素值（0-255）
			r, g, b, _ := resized.At(x, y).RGBA()
			rr, rg, rb := float32(r>>8), float32(g>>8), float32(b>>8)

			// Rescale: [0, 255] → [0, 1] (配置的 rescale_factor=1/255)
			rr /= 255.0
			rg /= 255.0
			rb /= 255.0

			// Normalize: (x - mean) / std (配置的 mean=0.5, std=0.5 → 最终范围 [-1, 1])
			rr = (rr - 0.5) / 0.5
			rg = (rg - 0.5) / 0.5
			rb = (rb - 0.5) / 0.5

			// 按 CHW 顺序填充数据
			// R通道: 0*H*W + y*W + x
			// G通道: 1*H*W + y*W + x
			// B通道: 2*H*W + y*W + x
			tensor[y*targetW+x] = rr                   // R
			tensor[targetH*targetW+y*targetW+x] = rg   // G
			tensor[2*targetH*targetW+y*targetW+x] = rb // B
		}
	}

	return tensor, []int64{1, 3, int64(targetH), int64(targetW)}, nil
}

// toRGBA 将任意图像转换为 RGBA 格式
func toRGBA(src image.Image) *image.RGBA {
	bounds := src.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, src, bounds.Min, draw.Src)
	return rgba
}
