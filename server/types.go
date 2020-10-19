package main

const (
	DeployTypeNew = "new"
	DeployTypeFwd = "fwd"
	DeployTypeLM  = "lm"
)

type Request struct {
	Method string `json:"method"`
	Deploy struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		NewApp struct {
			Image string `json:"image"`
			Port  struct {
				In  int `json:"in"`
				Ext int `json:"ext"`
			} `json:"port"`
			Env map[string]string `json:"env"`
		} `json:"newApp"`
		Fwd struct {
			SrcAddr string `json:"srcAddr"`
			Port    struct {
				In  int `json:"in"`
				Ext int `json:"ext"`
			} `json:"port"`
		} `json:"fwd"`
		LM struct {
			SrcAddr string `json:"srcAddr"`
			Port    struct {
				In  int `json:"in"`
				Ext int `json:"ext"`
			} `json:"port"`
		} `json:"lm"`
	} `json:"deploy"`
	Remove struct {
		Name string `json:"name"`
	} `json:"remove"`
	Checkpoint struct {
		Phase string `json:"phase"`
		Name  string `json:"name"`
	} `json:"_checkpoint"`
}

type Response struct {
	Ok  bool   `json:"ok"`
	Msg string `json:"msg"`
}
