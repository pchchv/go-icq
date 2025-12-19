package state

type sessionSlot struct {
	sess    *Session
	removed chan bool
}
