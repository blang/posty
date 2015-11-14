package model

type Model interface {
	PostPeer() PostPeer
	UserPeer() UserPeer
}
