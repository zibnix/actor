package actor

import "sync"

// A ptr to the zero value of an Actor is a valid instantiation:
//
//	actor := &Actor{}
//
// Teach the *Actor about various actions it should know how
// to perform in isolation with the pkg level Teach function.
type Actor struct {
	actionlk sync.Mutex
	wg       sync.WaitGroup
	quit     chan struct{}
}

// Not parallel safe with itself or a Teach call with the same *Actor.
func (a *Actor) Shutdown() {
	if a.quit != nil {
		close(a.quit)
	}
	a.wg.Wait()
}

// An action. You define the actions on some shared data in terms
// of (I)nput and (O)utput types that this pkg also doesn't need
// to know about. This pkg just makes sure those actions run in
// isolation using a rough actor pattern.
type Action[I, O any] interface {
	Act(I) O
}

// The ActionFunc type is an adapter to allow the use of
// ordinary functions as Actions. If f is a function
// with the appropriate signature, ActionFunc[I, O](f) is an
// Action that calls f.
type ActionFunc[I, O any] func(i I) O

func (f ActionFunc[I, O]) Act(i I) O {
	return f(i)
}

// Not parallel safe with itself or an Actor.Shutdown call on the same *Actor.
// Call this once and only once for each action the actor should know how to perform.
// The returned function gives you the ability to interact with the actor by submitting
// a type, and receiving a read-only chan on which to listen for the response.
// The returned function is parallel safe with other returns from Teach, but these
// returns are really the only functions in this pkg that make that promise.
// The <-chan O here should be read from every time, or you will get orphaned
// goroutines waiting to write.
func Teach[I, O any](actor *Actor, action Action[I, O]) func(I) <-chan O {
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
		act(actor, action, c)
	}()

	return func(i I) <-chan O {
		ochan := make(chan O)

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

func act[I, O any](actor *Actor, action Action[I, O], c chan struct {
	I     I
	Ochan chan O
}) {
	for {
		select {
		case s := <-c:
			actor.actionlk.Lock()
			o := action.Act(s.I)
			actor.actionlk.Unlock()

			go func() {
				select {
				case s.Ochan <- o:
				case <-actor.quit:
				}
			}()
		case <-actor.quit:
			return
		}
	}
}
