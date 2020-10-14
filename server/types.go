package main

type Request struct {
	Op     string `json:"op"`
	Create struct {
		Name      string            `json:name`
		CreateApp bool              `json:createApp`
		Image     string            `json:"image"`
		Port      int               `json:port`
		ExtPort   int               `json:extPort`
		Env       map[string]string `json:env`
	} `json:create`
	Delete struct {
		Name string `json:name`
	} `json:delete`
	Migrate struct {
		Name    string `json:name`
		SrcAddr string `json:srcAddr`
	} `json:migrate`
	Checkpoint struct {
		Name   string `json:name`
	} `json:_checkpoint`
}

type Response struct {
	Ok  bool   `json:ok`
	Msg string `json:msg`
}
