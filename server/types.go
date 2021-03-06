package main

const (
	DeployTypeNew   = "new"
	DeployTypeFwd   = "fwd"
	DeployTypeLM    = "lm"
	DeployTypeFwdLM = "fwdlm"
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
			Image   string `json:"image"`
			SrcAddr string `json:"srcAddr"`
			SrcName string `json:"srcName"`
			Port    struct {
				In  int `json:"in"`
				Ext int `json:"ext"`
			} `json:"port"`
			DstAddr   string            `json:"dstAddr"`
			Env       map[string]string `json:"env"`
			BwLimit   int               `json:"bwLimit"`
			Iteration int               `json:"iteration"`
		} `json:"lm"`
		FwdLM struct {
			Image   string `json:"image"`
			SrcAddr string `json:"srcAddr"`
			SrcName string `json:"srcName"`
			SrcPort int    `json:"srcPort"`
			Port    struct {
				In  int `json:"in"`
				Ext int `json:"ext"`
			} `json:"port"`
			DstAddr   string            `json:"dstAddr"`
			Env       map[string]string `json:"env"`
			BwLimit   int               `json:"bwLimit"`
			Iteration int               `json:"iteration"`
			DataRate  int               `json:"dataRate"`
		} `json:"fwdlm"`
	} `json:"deploy"`
	Remove struct {
		Name string `json:"name"`
	} `json:"remove"`
	DumpStart RequestDumpStart `json:"_startDump"`
}

type RequestDumpStart struct {
	Name    string `json:"name"`
	DstAddr string `json:"dstAddr"`
	BwLimit int    `json:"bwLimit"`
}

type Response struct {
	Ok  bool   `json:"ok"`
	Msg string `json:"msg"`
}
