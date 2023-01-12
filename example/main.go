package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/zibnix/actor"
)

// A Map adds methods to the builtin map
// that can be converted to Acts. An Act
// just takes one type and returns another.
//
//	type Act[I, O] func(I) O
type Map[K comparable, V any] map[K]V

func (m Map[K, V]) Get(key K) V {
	return m[key]
}

// If a return isn't necessary, the empty struct will do
func (m Map[K, V]) Del(key K) (s struct{}) {
	delete(m, key)
	return
}

func (m Map[K, V]) Put(kv KeyVal[K, V]) (s struct{}) {
	m[kv.Key] = kv.Val
	return
}

type KeyVal[K comparable, V any] struct {
	Key K
	Val V
}

var calls int

func init() {
	flag.IntVar(&calls, "v", 11, "upper bound for how many values to write to the map: [1, v)")
	flag.Parse()
}

func main() {
	var a actor.Actor

	m := make(Map[int, int])

	// these put and get functions are safe for parallel use.
	get := actor.Read(&a, m.Get)
	put := actor.Write(&a, m.Put)
	del := actor.Write(&a, m.Del)

	// Acts do not block, they create a goroutine that blocks

	// Acts return a chan that you block on while reading from
	// or safely ignore if, like with put and del, the return
	// is meaningless
	for i := 1; i < calls; i++ {
		// initialize the map vals to -1
		// so deletes are more obvious
		put(KeyVal[int, int]{i, -1})
	}

	// give the initialization some time to run
	time.Sleep(time.Millisecond)

	// now we'll write some values, while trying to delete them
	for i := 1; i < calls; i++ {
		// interleaving writes and deletes
		put(KeyVal[int, int]{i, i})
		del(i)
	}

	// now let's read
	// we should see some competition between writes and deletes
	var wg sync.WaitGroup

	fmt.Println("Interleaving Writes and Deletes:")
	for i := 1; i < calls; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// blocking read from the chan
			v := <-get(i)

			fmt.Printf("%v: %v\n", i, v)
		}(i)
	}

	wg.Wait()

	fmt.Println("\nInterleaving Reads and Writes:")
	for i := 1; i < calls; i++ {
		wg.Add(1)
		put(KeyVal[int, int]{i, 999})
		go func(i int) {
			defer wg.Done()
			v := <-get(i)
			fmt.Printf("%v: %v\n", i, v)
		}(i)

	}

	wg.Wait()
}
