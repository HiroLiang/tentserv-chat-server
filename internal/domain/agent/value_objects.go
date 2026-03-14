package agent

type ID int64

type Type string

const (
	Local  Type = "local"
	Remote Type = "remote"
	Hybrid Type = "hybrid"
)

type Status string

const (
	Available    Status = "available"
	Maintaining  Status = "maintaining"
	Discontinued Status = "discontinued"
	StatusError  Status = "error"
)

type Engine string

const (
	GGUF   Engine = "gguf"
	ONNX   Engine = "onnx"
	API    Engine = "api"
	Cloud  Engine = "cloud"
	MLC    Engine = "mlc"
	WebGPU Engine = "webGPU"
)
