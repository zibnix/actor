# Acctor
A generic lib for doing stuff in isolation via an actor-ish model.

### Why?

This lib has its own take on Actor model.

Not as fully featured as compared to alternatives, but also not introducing many
new or library-specific concepts or types, beyond language fundamentals. This lib
aims for minimalism, both in terms of lines of code and in terms of mental overhead.

## Example

See: example/main.go

Add some methods to the builtin map, under a new type, and use with the actor lib
to make it parallel safe:

```go
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
```

Create the parallel-safe functions for manipulating the map, and use
them wherever you want from here:

```go
var a actor.Actor

m := make(Map[K, V])

get := actor.TeachRead(&a, m.Get)
put := actor.TeachWrite(&a, m.Put)
del := actor.TeachWrite(&a, m.Del)

// put does not block, and we don't care about its return
put(KeyVal[K, V]{x, y})

// sometime later... do the read and block on its return
v := <-get(k)

// del does not block, and we don't care about its return
del(k)
```
