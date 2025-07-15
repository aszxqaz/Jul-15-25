package handler

type Error struct {
	Error string `json:"error"`
}

func NewError(message string) Error {
	return Error{
		Error: message,
	}
}
