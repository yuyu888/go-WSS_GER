package model

type Message struct {
	Body string
	DeviceId string
	Uid string
}

type Reply struct {
	Status int
	Data interface{}
}