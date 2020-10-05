package main

type MessageBody struct {
	Udid string `json:"udid"`
	Code int    `json:"code"`
	Text string `json:"text"`
}

type MessageRunWda struct {
	Type string      `json:"type"`
	Data MessageBody `json:"data"`
}

func NewMessageRunWda(udid string, code int, text string) MessageRunWda {
	return MessageRunWda{
		Type: "run-wda",
		Data: MessageBody{
			Udid: udid,
			Code: code,
			Text: text,
		},
	}
}
