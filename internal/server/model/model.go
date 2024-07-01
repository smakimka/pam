package model

type TokenData struct {
	ID    int
	Value string
}

type UserData struct {
	ID       int
	Username string
	Pwd      []byte
}

type Data struct {
	ID     int
	UserID int
	Name   string
	Kind   int
	Bytes  []byte
}

type ContextKey string

var UserID ContextKey = "userID"
var AuthToken ContextKey = "authToken"
