package main

type UserPublicInfo struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	PublicKey string `json:"publicKey"`
}

type User struct {
	PublicInfo UserPublicInfo `json:"userPublicInfo"`
}
