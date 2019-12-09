// Package state assists in communicating with stateful go routines through channels. You simple declare a type SendToStateChan,
// have the go routine listen on that channel, and send data through that channel with the Send function. The SendToStateChan
// is a channel of SendInterface, so in the goroutine you can get the object sent with the Get() method, and return a response
// over the channel with the Return method
//
// To help with casting the interfaces sent over the channel their are helper functions in the cast.go file.
package state

// SendToStateChan is a chan SendInterface.
type SendToStateChan chan SendInterface

// SendInterface implements methods to easily communicate with a stateful goroutine
type SendInterface interface {
	Get() interface{}
	Return(interface{})
	Close()
}

type sendStruct struct {
	obj interface{}
	r   chan interface{}
}

func (s *sendStruct) Get() interface{} {
	return s.obj
}

func (s *sendStruct) Return(obj interface{}) {
	s.r <- obj
}

func (s *sendStruct) Close() {
	close(s.r)
}

// Send creates a SendInterface from the given interface and sends it over the channel. It waits for a response from the
// channel and returns that
func Send(ch SendToStateChan, obj interface{}) interface{} {
	r := make(chan interface{})
	ch <- &sendStruct{
		obj: obj,
		r:   r}
	return <-r
}
