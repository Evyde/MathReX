package model_controller

import (
	onnxruntime "github.com/yalue/onnxruntime_go"
)

type Encoder struct {
	session *onnxruntime.DynamicAdvancedSession
	//inputTensor  *onnxruntime.Tensor[float32]
	//outputTensor *onnxruntime.Tensor[float32]
}

var (
	encoder *Encoder
)

func NewEncoder(modelPath string, useCoreML bool, useCUDA bool) (*Encoder, error) {
	if encoder == nil {
		encoder = &Encoder{}
		var err error

		inputName := []string{"pixel_values"}
		outputName := []string{"last_hidden_state"}

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

		//encoder.inputTensor, err = onnxruntime.NewEmptyTensor[float32](onnxruntime.NewShape(1, 3, 384, 384))
		//
		//if err != nil {
		//	return nil, err
		//}
		//
		//encoder.outputTensor, err = onnxruntime.NewEmptyTensor[float32](onnxruntime.NewShape(1, 578, 384))
		//
		//if err != nil {
		//	return nil, err
		//}

		encoder.session, err = onnxruntime.NewDynamicAdvancedSession(
			modelPath,
			inputName,
			outputName,
			options,
		)

		if err != nil {
			return nil, err
		}
	}
	return encoder, nil
}

func (encoder *Encoder) Run(inputTensor []onnxruntime.Value) ([]float32, error) {
	r := []onnxruntime.Value{nil}
	err := encoder.session.Run(
		inputTensor,
		r,
	)
	defer r[0].Destroy()
	if err != nil {
		return nil, err
	}

	labelTensor := r[0].(*onnxruntime.Tensor[float32])
	predictedLabels := labelTensor.GetData()

	return predictedLabels, nil
}
