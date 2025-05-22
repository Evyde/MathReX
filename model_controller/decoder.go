package model_controller

import (
	"fmt"
	onnxruntime "github.com/yalue/onnxruntime_go"
)

type DecoderConfig struct {
	BosTokenID          int64
	EosTokenID          int64
	MaxLength           int
	VocabSize           int
	EncoderSeqLen       int // 578（来自encoder输出）
	DecoderStartTokenID int64
}

type Decoder struct {
	session *onnxruntime.DynamicAdvancedSession
	config  *DecoderConfig
}

var (
	decoderInst *Decoder
)

func NewDecoder(modelPath string, useCoreML bool, useCUDA bool) (*Decoder, error) {
	if decoderInst == nil {
		decoderInst = &Decoder{}
		var err error
		config := &DecoderConfig{
			BosTokenID:          1,
			EosTokenID:          2,
			MaxLength:           512,
			VocabSize:           1200,
			EncoderSeqLen:       578,
			DecoderStartTokenID: 2,
		}

		decoderInst.config = config

		options, err := onnxruntime.NewSessionOptions()

		if err != nil {
			return nil, err
		}

		if useCoreML {
			err := options.AppendExecutionProviderCoreML(0)
			if err != nil {
				return nil, err
			}
		}

		if useCUDA {
			cudaOptions, err := onnxruntime.NewCUDAProviderOptions()
			if err != nil {
				return nil, err
			}
			err = options.AppendExecutionProviderCUDA(cudaOptions)
			if err != nil {
				return nil, err
			}
		}

		decoderInst.session, err = onnxruntime.NewDynamicAdvancedSession(
			modelPath,
			[]string{"input_ids", "encoder_hidden_states"},
			[]string{"logits"},
			options,
		)
		if err != nil {
			return nil, err
		}
	}
	return decoderInst, nil
}

func (d *Decoder) Close() {
	onnxruntime.DestroyEnvironment()
}

func (d *Decoder) decodeStep(
	inputIDs []int64,
	encoderStates []float32, // [batch][seq_len][hidden_size]
) ([]float32, error) {
	// 创建输入张量
	inputTensor, _ := onnxruntime.NewTensor(
		onnxruntime.NewShape(1, int64(len(inputIDs))), // [batch=1, seq_len]
		inputIDs,
	)
	defer inputTensor.Destroy()

	encoderTensor, _ := onnxruntime.NewTensor(
		onnxruntime.NewShape(1, int64(d.config.EncoderSeqLen), 384), // [1, 578, 384]
		encoderStates,
	)
	defer encoderTensor.Destroy()

	output := []onnxruntime.Value{nil}

	// 执行推理
	err := d.session.Run(
		[]onnxruntime.Value{inputTensor, encoderTensor},
		output,
	)

	if err != nil {
		return nil, err
	}

	// 获取logits输出
	logitsTensor := output[0].(*onnxruntime.Tensor[float32])

	defer logitsTensor.Destroy()

	return logitsTensor.GetData(), nil
}

func (d *Decoder) Generate(encoderOut []float32) ([]uint32, error) {
	// 初始化生成序列
	generatedIDs := []int64{d.config.DecoderStartTokenID}

	for len(generatedIDs) <= d.config.MaxLength {
		// 执行单步解码
		logits, err := d.decodeStep(generatedIDs, encoderOut)

		if err != nil {
			return int64ToUint32Slice(generatedIDs), err
		}

		// 获取最后一个位置的logits
		lastLogits := extractLastLogits(logits, len(generatedIDs), d.config.VocabSize)

		// 选择下一个token
		nextID := argmax(lastLogits)
		generatedIDs = append(generatedIDs, nextID)

		// 终止条件
		if nextID == d.config.EosTokenID {
			break
		}
	}

	for _, i := range generatedIDs {
		fmt.Printf("%d ", i)
	}

	return int64ToUint32Slice(generatedIDs), nil
}

// 将int64切片转换为uint32切片
func int64ToUint32Slice(input []int64) []uint32 {
	output := make([]uint32, len(input))
	for i, v := range input {
		if v < 0 {
			continue
		}
		output[i] = uint32(v)
	}
	return output
}

// 提取最后一个token的logits
func extractLastLogits(fullLogits []float32, seqLength, vocabSize int) []float32 {
	startIdx := (seqLength - 1) * vocabSize
	return fullLogits[startIdx : startIdx+vocabSize]
}

// 贪心选择
func argmax(logits []float32) int64 {
	maxIdx := 0
	for i := 1; i < len(logits); i++ {
		if logits[i] > logits[maxIdx] {
			maxIdx = i
		}
	}
	return int64(maxIdx)
}
