package model

// Model defines a basic model consisting of two entities `post` and `user`.
type Model interface {
	PostPeer() PostPeer
	UserPeer() UserPeer
}
