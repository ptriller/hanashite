package users

type User struct {
	Name   string `json:"name"`
	Pubkey []byte `json:"pubkey"`
}
