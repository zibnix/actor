package actor

import "sync"

// The zero value of an Actor is a valid instantiation:
//
//	var a actor.Actor
//
// Teach the *Actor about various acts it should know how
// to perform in isolation with the pkg level Teach function.
//
//	read := TeachRead(&a, rfunc)
//	write := TeachWrite(&a, wfunc)
type Actor struct {
	actlk sync.RWMutex
	wg    sync.WaitGroup
	quit  chan struct{}
}

// Not parallel safe with itself or a Teach call with the same *Actor.
func (a *Actor) Shutdown() {
	if a.quit != nil {
		close(a.quit)
	}
	a.wg.Wait()
}

// An act. You define the acts on some shared data in terms
// of (I)nput and (O)utput types that this pkg also doesn't need
// to know about. This pkg just makes sure those acts run in
// isolation using a rough actor pattern.
type Act[I, O any] func(I) O

type RW int

const (
	Read RW = iota
	Write
)

// Teach an *Actor how to perform a read act.
// The pkg level Teach functions aren't parallel safe with each other or an
// Actor.Shutdown call on the same *Actor.
func Reader[I, O any](actor *Actor, act Act[I, O]) func(I) <-chan O {
	return Teach(actor, act, Read)
}

// Teach an *Actor how to perform a write act.
// The pkg level Teach functions aren't parallel safe with each other or an
// Actor.Shutdown call on the same *Actor.
func Writer[I, O any](actor *Actor, act Act[I, O]) func(I) <-chan O {
	return Teach(actor, act, Write)
}

// Teach functions launch an actor that runs the provided Act in isolation.
// You can call Teach functions more than once for the same Act, which would
// give the ability to have multiple readers/writers for the same Act, if needed.
// You would still need to keep track of the returned functions, and spread
// calls across them.
// The returned function has the ability to interact with the actor by submitting
// a type, and receiving a read-only chan on which to listen for the response.
// The returned function is parallel safe with other returns from Teach.
// Reads can occur in parallel with each other.
// Writes are excluded from occurring in parallel with other writes and reads.
// Teach functions aren't methods because methods cannot have type params, and
// each Act should be able to supply its own types.
//
// Currently, the returned function does not block when called, and does not need
// to be called with the `go` keyword.
//
// Instead, the returned function could itself return a chan and a function to do
// the communication with the actor, rather than starting a goroutine for that
// communication. This would allow the caller to control when/if a new goroutine
// needs to be run for communication, which would perhaps be more in line with
// recommendations for API behavior.
// As it is now, the signature is slightly simpler just returning the chan, and
// this enables reading from the chan directly without creating a named variable
// in importing code. If the potential extra overhead of that automatically
// started goroutine is an issue, and you'd rather have the option of blocking on
// the write in your own goroutine, feel free to let me know or fork.
func Teach[I, O any](actor *Actor, act Act[I, O], rw RW) func(I) <-chan O {
	if actor.quit == nil {
		actor.quit = make(chan struct{})
	}

	c := make(chan struct {
		I     I
		Ochan chan O
	})

	actor.wg.Add(1)
	go func() {
		defer actor.wg.Done()
		action(actor, act, rw, c)
	}()

	return func(i I) <-chan O {
		// buffered so that the actor can write without blocking
		// or spinning up another goroutine
		ochan := make(chan O, 1)

		go func() {
			select {
			case c <- struct {
				I     I
				Ochan chan O
			}{
				I:     i,
				Ochan: ochan,
			}:
			case <-actor.quit:
			}
		}()

		return ochan
	}
}

func action[I, O any](actor *Actor, act Act[I, O], rw RW, c chan struct {
	I     I
	Ochan chan O
}) {
	var lock, unlock func()

	switch rw {
	case Read:
		lock = actor.actlk.RLock
		unlock = actor.actlk.RUnlock
	case Write:
		fallthrough
	default:
		lock = actor.actlk.Lock
		unlock = actor.actlk.Unlock
	}

	for {
		select {
		case s := <-c:
			lock()
			o := act(s.I)
			unlock()

			s.Ochan <- o
		case <-actor.quit:
			return
		}
	}
}
