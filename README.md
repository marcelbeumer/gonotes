# notes-in-go

Partial port of personal notes system from TS to Go, for learning purposes. Work in progress.

# What I like about go

- Relatively simple coming from TypeScript
- Most things easy to find in the docs and examples, otherwise elsewhere online
- With some exceptions, I overall like the relatively calm and clean syntax, despite being c-style
- Convention of capital identifiers exporting
- Module conventions like `import "foo/bar/name"` and then `name.Method()`
- Seems like API's are often straight forward and stdlib provides a lot of basics

# What I dislike about go

## Language server did not pick up on new files and dependencies

Could have been something wrong in my neovim LSP + gopls setup, but I had to restart neovim every now and then.

## Unused variables and imports are compile errors

It sucks you can't compile when you're moving around code without cleaning everything up, and it also sucks your editor (neovim in my case) always shows errors, instead of marking them as warnings. I do agree with the Go FAQ that "if it's worth complaining about it's worth fixing the code" but I don't agree that's a reason to not having warnings.

(What's next, not allowing a build when the tests fail?)

## Type-system can not distinguish between initialized and uninitialized maps and arrays

Consumers of your `func () string[]` may get a result value that `== nil` when the function returns an uninitialized array and otherwise not. It's probably idiomatic to just not do a nil-check on those return values but, a language that brags about being restrictive and enterprise-safe on purpose leaves a hole where an implementation detail
may introduce bugs further down.

```go
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

```go
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

```go
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

```go
	e := example.New()
	e.Something() // fine

	e2 := example.Example{}
	e2.Something() // panic
```

## No sum type and no great alternatives either

I wanted to do something like `type MetaField = StringField | TimeField | IntField` but you can not in Go. Instead, there are ways to workaround but none of them seem satisfying.

Idiomatic seems to be:

```go
type MetaField interface {
	sealed()
	Method()
}

type StringField struct { }
func (f *StringField) sealed() {}
func (f *StringField) Method() {}

type TimeField struct { }
func (f *TimeField) sealed() {}
func (f *TimeField) Method() {}
```

and then use type switches to figure out if a specific `Metafield` is one of the "subtypes". Apart from the bulky switch statements (compared to more elegant pattern matching), those switch statement can not be exhaustive because switch does not support that, and no one except the package itself knows which variants belong to `Metafield`.

In some cases you could solve the sum type by just using one struct that can contain all variants

```go
type MetaField struct {
  Time *time.Time
  Int  *int
  String *string
}
```

Or something like that.

## Implicit interface implementations

You just implement the methods for a struct to satisfy an interface without being explicit about it. On one hand it's great to have some sort of structural typing here, but on the other hand, it's annoying you can not make explicit if you wanted to, because instead of getting clear error messages where you are trying to implement a certain interface, you now get more cryptic messages where you are trying to use them.

## No generic programming, casting to `interface{}`

There is no `map` and `reduce` and so on, but there's also no way to properly implement then without support for generic programming, through generics or otherwise.

Instead, it seems people often cast things to `interface{}` to write generic code, then using type switches or reflection to "recover" from that. Or, just not write generic code.

## No optional function parameters

You could make multiple functions for different variants, or pass an options struct of sorts, but both are pretty annoying imo.
