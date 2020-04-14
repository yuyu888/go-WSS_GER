package model

type Message struct {
	Content string
	Wssid string
}

type Reply struct {
	Status int
	Data interface{}
}