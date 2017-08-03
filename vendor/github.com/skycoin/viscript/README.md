# viscript

[![GoDoc](
	https://godoc.org/github.com/skycoin/viscript?status.svg)](
		https://godoc.org/github.com/skycoin/viscript)
[![Go Report Card](
	https://goreportcard.com/badge/github.com/skycoin/viscript)](
		https://goreportcard.com/report/github.com/skycoin/viscript)


Dependencies
============

Dependencies are managed with [dep](https://github.com/golang/dep).

To install `dep`:

```sh
go get -u github.com/golang/dep
```

`dep` vendors all dependencies into the repo.

If you change the dependencies, you should update them as needed with `dep ensure`.

Use `dep help` for instructions on vendoring a specific version of a dependency, or updating them.

When updating or initializing, `dep` will find the latest version of a dependency that will compile.

After adding a new dependency (with `dep ensure`), run `dep prune` to remove any unnecessary subpackages from the dependency.

**Warning:** `dep prune` will remove necessary C files from `github.com/go-gl/glfw/v3.2/`.
In order to preserve these, do `git checkout vendor/github.com/go-gl/glfw/v3.2/` after `dep prune`.

Examples:

Initialize all dependencies:

```sh
dep init
git add Gopkg.toml Gopkg.lock vendor/
git commit -m "dep init"
dep prune
git checkout vendor/github.com/go-gl/glfw/v3.2
git add vendor/
git commit -m "dep prune"
```

Update all dependencies:

```sh
dep ensure -update -v
dep prune
git checkout vendor/github.com/go-gl/glfw/v3.2
git add vendor/
git commit -m "dep ensure update"
```

Add a single dependency (latest version):

```sh
dep ensure github.com/foo/bar
dep prune
git checkout vendor/github.com/go-gl/glfw/v3.2
git add vendor/
git commit -m "dep ensure github.com/foo/bar"
```

Add a single dependency (more specific version), or downgrade an existing dependency:

```sh
dep ensure github.com/foo/bar@tag
dep prune
git checkout vendor/github.com/go-gl/glfw/v3.2
git add vendor/
git commit -m "dep ensure github.com/foo/bar@tag"
```

[//]: # (OpenGL, GLFW and other dependencies:)

[//]: # (github.com/go-gl/gl/v{3.2,3.3,4.1,4.4,4.5}-{core,compatibility}/gl)

[//]: # (go get github.com/go-gl/gl/v2.1/gl)

[//]: # (go get github.com/go-gl/glfw/v3.2/glfw)

[//]: # (go get github.com/skycoin/skycoin)

Building on Debian
=======
```
sudo apt-get install libxi-dev
sudo apt-get install libgl1-mesa-dev
sudo apt-get install libxrandr-dev
sudo apt-get install libxcursor-dev
sudo apt-get install libxinerama-dev
```

Spec
====

Macros + Reflection
Statically typed scheme

Define types
- i32 (int 32)
- u8 (uint 8)
- []byte (byte array, all byte arrays are fixed length)

Define operators
- i32_add
- i32_sub
- i32_mult
- i32_div

Syntax
- (i32_add 3, 5)

A function
- has tuple of inputs, tuple of outputs
- i32_add has type (i32, i32) (i32); takes in two i32 and returns one i32
- a function has a list of statements

A struct
- a tuple of types
- (i32, i32) is a tuple of two i32

(def_struct Tag,
	tag []byte
	uuid [16] byte
)

A union
- means "has to choose A or B"
- enumerates possibilities
- a choice of types

A function has an array of "statements" or "blocks" (an array of statements) or a control flow (if/then, for)

(def_func, NAME, (u32 a, u32 b), (u32),
	(u32_add a, b)
	)

Type types of operators
- functions on (can modify object)
- functions of (cannot modify object)
- functions on and functions of, should be in different colors

The type of an object is a type
- the type of an object is a struct, defining the type

(def_func_on ...)
(def_func_of ...)

def_func_of cannot call functions def_func_on
- can only read program, cannot modify

(def_var_m i32 x)
- defines new entity x

(def_var_imutable i32 x)
- defines new variable, which cannot be changed after creation

An "assert" is something that must be true

An "affordance" is something that CAN be done to an object
- all affordances, must be enumerable
- a user must be able to select each possible action, from a list

A "restriction" is something that CANNOT be done to an object
- restrictions must be checked and the affordance list filtered
- Some types of restrictions, can be applied to a program as an operation, to remove an affordance
- an example, is that structs/functions are defined, then the affordance for modifying them is removed (allowing compilation or simplification to a static binary)

A "context" is current state
- list of functions
- list of structs
- list of defined things
- the context contains the current module, the stack, the current line and function
- current function the program is on
- current statement the program is on
- list of variables in the current scope
- list of functions in the current module
- list of variables defined in a module

reflection
- each type has a list of functions and operators on it
- reflection on an object is a func_of the object meta_type
- modifying or extending an object is a func_on the object meta_type

(is_def x)
- returns if thing is defined

(is_type x, type)
- determines of object is of type x

(type_of x)
- returns type

A "choice" is a place where A or B (where program can choose A or B)
- unions are for objects (structs)
- choices are for code
- a choice has a list of preconditions (things that must be true for choice to be made) and a list of statements (which can only be chosen if the preconditions are met)
- a choice may have a signature (if it returns an object)
- a choice could be modeled as a special type of function, that returns something
- the choice operator, is a special function, that takes in the current context (state of program)

(def_func, NAME, (uint a, uint b), (uint),
	(choice,
		(true, (u32_add a, b)),
		(true, (u32_add b,a)))
	)
)

UIDs
- struct { text_tag []byte , uuid [16]byte}
- all variables, functions, modules have 128 bit UIDS
- a UUID has a text "text_tag", or keyboard/display name
- The UUID is used to look up the object in a table

Modules
- all code occurs in modules
- a module has a name
- a module contains functions and function definitions
- a module contains structs and struct definitions

(as x y)
(as x z)
- another way of writing choice operator
- see XL programming language
- when conditions occurs, x can be choice Y or Z
- as operator, defines choices, that are not bound to a context or object
- the as operator, defines the conditions, when an affordance is available

- The program itself is an object (a struct)
- The program begins with a default object (the null object)
- The program starts with a series of actions that can be applied to it (affordances)
- The program is built up, by a series of operators applied to it (affordances)

Programs accept and emit only length prefixed messages
- there is a function for checking if there is an incoming length prefixed message
- there is a function for emitting a length prefixed message
- there is a function for receiving the length prefixed message from the queue
- there is a function for halting the program, until a length prefixed message is available

A program can spawn, isolated sub-programs that it can only communicate to over length prefixed messages

An "agent" is a program that can apply affordances
- the agent program reads the curent affordances on objects and applies them
- agents are often restricted to a subset of objects and a subset of affordances

A "behavior" is a set of criteria or goals or states, that the agent attempts to maintain

Reflection
- the fields and functions on a struct can be enumerated
- the signature and body of a function can be enumerated
- the list of structs, variables and objects in a given module can be enumerated
- the list of modules, imported by a given module can be enumerated
- the list of local scope variables, in a given function/context/stack frame can be enumerated
- the types of each variable, can be enumerated

Reflection on dependecy graphs

Define a program object
- then apply its operators on it, to construct the program

---

!!!
Crash logs
- overlay all crashes on the source code
- find and trace crash logs

Interactions between classes/functions
- graph all interactions


====

atomic types (int32, uin32, []byte)
operatiosn on atomic types (uint32.add a b)
structs
functions
modules
type signatures

Atomic types
- ints
- byte arrays
- "Type" objects

====

A function is a
1> name (text)
2> input tuple (name, type) pair list
3> output tuple (list of types/signatures returned)
4> an array of lines/statements

A struct is a
1> name
2> list of (name, type) pairs
3> later list of functions on the struct but ignore this for now

A module is
0> The name of the module (string)
1> A list of modules imported by the current module
2> A list of structs defined in the current module
3> A list of function defined in the current module
4> A list of variables at the global scope of the module

===

Note:
- modules
- structs
- functions

Should all have unique ids, to be used as references
- unique IDs can be 64 bit (effectively pointers to def)

A function is

struct Function {
	name []byte
	input []struct{[]byte name, type Type)} //name/type pair array
	output []struct{[]byte name, type Type} //optional name
	lines []Expressions
}

A struct is

struct Struct {
	name []bytes
	fields []struct{[]byte name, type Type)} //name/type pair aray
}

A module is

struct Module {
	name []bytes
	module_imports []*Modules
	module_functions []*function
	module_structs []*structs
}

Each of these is written as S notation

(def_func Name (in...) (out...) (expression_array...) )


System Level Enumeration
========================

System Level Enumerations
- give me a list of nodes I controll
- give me list of programs running on a node
- give me a list of channels (communication channels) between nodes

- deploy a task on a node
- shutdown task on a node

- get CPU/ ram usage, etc

Language Level Enumeration
==========================

In a given line of source code
- enumerate the variables (types, name) in the current scope
- enumerate the variables (types, name) passed into the current function
- enumerate the variables, modules, functions that can be called from the current line/scope
- enumerate the variables in the local scope
- enumerate the variables passed into a function
- enumerate the variables at global, current module level
- enumerate the current modules that are imported in the current module

- enumerate the defined functions in the current module
- enumerate the defined global variables in the current module

(var x uint32) adds a new variable to the local scope

A function that enumerates the list of atomic/base types
A function that enumerates the list of defined types

Types
- A function that enumerates the list of atomic/base types
- A function that enumerates the list of defined types
- enumerate the fields of a type (struct)
- enumerate the functions OF a type (functions that do not modify its state, function of an instance)
- enumerate the functtions ON a type (functoins that change its state, functions ON a type instance)
- enumerate state (name, type) pairs for struct type and the functions on the tpe

- enumerate the dependencies on an object
-- example: What external functions, objects, modules are used by a particular function
-- what external functions, objects, modules are used by a line in a particular function
















