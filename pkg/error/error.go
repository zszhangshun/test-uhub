package error

type Message struct {
	Msg string `json:"message" binding:"required"`
}

func New(msg string) Message {
	return Message{
		Msg: msg,
	}
}

