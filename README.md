# notes-in-go

Partial port of personal notes system from TS to Go, for learning purposes

# What I like about go

# What I dislike about go

## Language server did not pick up on new files and dependencies

Could have been something wrong in my neovim LSP + gopls setup, but I had to restart neovim every now and then.

## Unused variables and imports are compile errors

It sucks you can't compile when you're moving around code without cleaning everything up, and it also sucks your editor (neovim in my case) always shows errors, instead of marking them as warnings. I do agree with the Go FAQ that "if it's worth complaining about it's worth fixing the code" but I don't agree that's a reason to not having warnings.

## Type-system can not distinguish between initialized and uninitialized maps and arrays

Consumers of your `func () string[]` may get a result value that `== nil` when the function returns an uninitialized array and otherwise not. It's probably idiomatic to just not do a nil-check on those return values but, a language that brags about being restrictive and enterprise-safe on purpose leaves a hole where an implementation detail
may introduce bugs further down.

```
func testA() []string {
	return make([]string, 0)
}

func testB() []string {
	var arr []string
	return arr
}

func testC() {
	a := testA()
	b := testB()
	if a == nil { // will not happen
		fmt.Println("a is nil")
	}
	if b == nil { // will happen
		fmt.Println("b is nil")
	}
}
```

Similarly, uninitialized maps type-check against initialized maps. As a result, you can not seem make safe assumptions about a map in a struct without doing a nil-check.

```
type example struct {
	stuff map[string]string
	name  string
}

func useExample(shouldPanic bool) {
	s := example{}  // new struct
	s.name = "name" // fine
	if shouldPanic {
		s.stuff["some"] = "xxx" // panic: assignment to entry in nil map
	} else {
		s.stuff = make(map[string]string)
		s.stuff["some"] = "xxx" // works!
	}
}
```

This has big implications. Let's say you have a package like so:

```
package example

type Example struct {
	Stuff map[string]string
	Name  string
}

func New() Example {
	e := Example{}                    // new struct
	e.Stuff = make(map[string]string) // better do this :)
	return e
}

func (e *Example) Something() {
	e.Stuff["some"] = "xxx" // may panic if not initialized
}
```

Now when something is using your package he/she needs to know if it's needed to use your `New` constructor function, or if it's safe to just create the struct.

```
	e := example.New()
	e.Something() // fine

	e2 := example.Example{}
	e2.Something() // panic
```
