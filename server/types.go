package main

type Request struct {
	Op     string `json:"op"`
	Create struct {
		Name     string `json:name`
		Image    string `json:"image"`
		Port     int    `json:port`
		NodePort int    `json:nodePort`
	} `json:create`
	Delete struct {
		Name string `json:name`
	} `json:delete`
}

type Response struct {
	Ok  bool   `json:ok`
	Msg string `json:msg`
}
