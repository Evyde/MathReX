package model_controller

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/dop251/goja"
	onnxruntime "github.com/yalue/onnxruntime_go"
)

var tk *Tokenizer
var katexJSCode []byte
var mathml2ommlJSCode []byte // For mathml2omml.js
var encoderModel *Encoder
var decoderModel *Decoder

func InitKaTeX(data []byte) {
	katexJSCode = data
	if len(katexJSCode) == 0 {
		log.Println("Warning: KaTeX JS data is empty during InitKaTeX.")
	}
}

// InitMathML2OMMLJS stores the mathml2omml.js code.
func InitMathML2OMMLJS(data []byte) {
	mathml2ommlJSCode = data
	if len(mathml2ommlJSCode) == 0 {
		log.Println("Warning: mathml2omml.js data is empty during Init.")
	}
}

func InitTokenizer(path string) error {
	t, err := NewTokenizer(path)
	if err != nil {
		return fmt.Errorf("failed to initialize tokenizer: %w", err)
	}
	tk = t
	return nil
}

func InitModels(encoderPath, decoderPath string) error {
	var err error
	encoderModel, err = NewEncoder(encoderPath, false, false)
	if err != nil {
		return fmt.Errorf("failed to initialize encoder: %w", err)
	}
	decoderModel, err = NewDecoder(decoderPath, false, false)
	if err != nil {
		return fmt.Errorf("failed to initialize decoder: %w", err)
	}
	log.Println("Encoder and Decoder initialized successfully.")
	return nil
}

func ProcessImagePrediction(imageData []byte, outputFormat string) (resultText string, resultTokens []uint32, err error) {
	if encoderModel == nil || decoderModel == nil {
		return "", nil, fmt.Errorf("models not initialized. Call InitModels first")
	}
	if tk == nil {
		return "", nil, fmt.Errorf("tokenizer not initialized. Call InitTokenizer first")
	}

	tmpFile, err := ioutil.TempFile("", "tempimage-*.png")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary image file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(imageData); err != nil {
		tmpFile.Close()
		return "", nil, fmt.Errorf("failed to write image data to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", nil, fmt.Errorf("failed to close temporary image file: %w", err)
	}

	fileForProcessing, err := os.Open(tmpFile.Name())
	if err != nil {
		return "", nil, fmt.Errorf("failed to reopen temporary image file for processing: %w", err)
	}
	defer fileForProcessing.Close()

	encoderData, encoderShape, err := PreprocessToModelFormat(fileForProcessing)
	if err != nil {
		return "", nil, fmt.Errorf("failed to preprocess image: %w", err)
	}

	inputTensor, err := onnxruntime.NewTensor(encoderShape, encoderData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create encoder input tensor: %w", err)
	}

	outputValue, err := encoderModel.Run([]onnxruntime.Value{inputTensor})
	if err != nil {
		return "", nil, fmt.Errorf("encoder run failed: %w", err)
	}

	tokens, err := decoderModel.Generate(outputValue)
	if err != nil {
		return "", nil, fmt.Errorf("decoder generation failed: %w", err)
	}
	log.Println("Generated tokens:", tokens)

	latex := tk.Decode(tokens)

	switch outputFormat {
	case "latex":
		resultText = latex
	case "mathml":
		mathml, errConv := convertLatexToMathML(latex)
		if errConv != nil {
			return "", tokens, fmt.Errorf("LaTeX to MathML conversion failed: %w", errConv)
		}
		resultText = mathml
	case "omml": // OMML re-enabled
		mathml, errConv := convertLatexToMathML(latex)
		if errConv != nil {
			return "", tokens, fmt.Errorf("LaTeX to MathML conversion failed: %w", errConv)
		}
		omml, errConv := convertMathMLToOMML(mathml)
		if errConv != nil {
			return "", tokens, fmt.Errorf("MathML to OMML conversion failed: %w", errConv)
		}
		resultText = omml
	default:
		return "", tokens, fmt.Errorf("invalid format: %s. Supported formats are latex, mathml, omml", outputFormat)
	}

	return resultText, tokens, nil
}

func convertLatexToMathML(latex string) (string, error) {
	if len(katexJSCode) == 0 {
		return "", fmt.Errorf("KaTeX JavaScript code has not been initialized or is empty")
	}
	vm := goja.New()
	_, err := vm.RunString(string(katexJSCode))
	if err != nil {
		return "", fmt.Errorf("failed to execute KaTeX JavaScript: %w", err)
	}
	err = vm.Set("latexInput", latex)
	if err != nil {
		return "", fmt.Errorf("failed to set latexInput variable in goja VM: %w", err)
	}
	script := `
		var mathmlString;
		try {
			mathmlString = katex.renderToString(latexInput, { output: "mathml", throwOnError: false });
		} catch (e) {
			mathmlString = '<math><merror><mtext>' + e.toString() + '</mtext></merror></math>';
		}
		mathmlString;
	`
	value, err := vm.RunString(script)
	if err != nil {
		return "", fmt.Errorf("failed to run KaTeX render script in goja: %w", err)
	}
	mathmlOutput := value.String()
	startIndex := strings.Index(mathmlOutput, "<math")
	if startIndex == -1 {
		return mathmlOutput, nil
	}
	endIndex := strings.LastIndex(mathmlOutput, "</math>")
	if endIndex == -1 || endIndex < startIndex {
		return mathmlOutput, nil
	}
	endIndex += len("</math>")
	return mathmlOutput[startIndex:endIndex], nil
}

// convertMathMLToOMML re-enabled with goja and mathml2omml.js
func convertMathMLToOMML(mathml string) (string, error) {
	if len(mathml2ommlJSCode) == 0 {
		return "", fmt.Errorf("mathml2omml.js code has not been initialized or is empty")
	}

	vm := goja.New()
	_, err := vm.RunString(string(mathml2ommlJSCode))
	if err != nil {
		return "", fmt.Errorf("failed to execute mathml2omml.js: %w", err)
	}

	err = vm.Set("mathmlInput", mathml)
	if err != nil {
		return "", fmt.Errorf("failed to set mathmlInput variable in goja VM: %w", err)
	}

	script := `
		var ommlString;
		try {
			if (typeof mml2omml === 'function') {
				ommlString = mml2omml(mathmlInput);
			} else if (typeof MathML2OMML !== 'undefined' && typeof MathML2OMML.mml2omml === 'function') {
				ommlString = MathML2OMML.mml2omml(mathmlInput);
			} else if (typeof MathML2OMML !== 'undefined' && typeof MathML2OMML.convert === 'function') { 
				ommlString = MathML2OMML.convert(mathmlInput);
			} else {
				throw new Error("mml2omml function (or MathML2OMML.mml2omml/convert) not found in global scope after loading script.");
			}
		} catch (e) {
			ommlString = "<!-- Error converting MathML to OMML: " + e.toString() + " -->";
		}
		ommlString;
	`
	value, err := vm.RunString(script)
	if err != nil {
		return "", fmt.Errorf("failed to run mathml2omml conversion script in goja: %w", err)
	}

	return value.String(), nil
}
