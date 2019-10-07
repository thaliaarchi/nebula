# Whitespace Introduction

The Whitespace language is defined by Edwin Brady and Chris Morris in
the original [Haskell interpreter source](https://web.archive.org/web/20150717140342/http://compsoc.dur.ac.uk/whitespace/download.php)
and the [language tutorial](https://web.archive.org/web/20150618184706/http://compsoc.dur.ac.uk/whitespace/tutorial.php).

## Grammar

The grammar of the language is very simple, consisting only of the
characters space, tab, and line feed. Instructions are denoted with
permutations of these characters, resembling a prefix tree. Integers are
an arbitrary-length series of binary digits written as spaces and tabs
and terminated with a line feed.

As the only recognized characters are invisible, programs are hard to
read and write, thus I have only written one program directly with this
syntax: [hello_world.ws](programs/hello_world.ws). Many authors instead
write in one of various "Whitespace Assembly" dialects that provide a
small level of abstraction. I borrow from the syntax used by the
[Whitelips IDE project](https://github.com/vii5ard/whitespace).

## Architecture

The Whitespace memory model consists of a stack and a heap. In the
stack, values are pushed once and accessed once and storage is
sequential. The heap is addressed dynamically, allowing for multiple
assignments and accesses at the same address.

Control flow consists of labels, unconditional and conditional jumps,
call, return, and exit. A control stack is used for call and return,
separate from the data stack.

## Instruction set

### Stack manipulation

Stack operations push, pop, or otherwise modify the data stack. Copy and
slide were added in Whitespace 0.3 to facilitate recursion.

| Command | Parameters | Stack | Meaning                                           |
| ------- | ---------- | ----- | ------------------------------------------------- |
| `push`  | Number     | -     | Push the number onto the stack                    |
| `dup`   | -          | -     | Duplicate the top item on the stack               |
| `copy`  | Number     | -     | Copy the nth item on the stack                    |
| `swap`  | -          | -     | Swap the top two items on the stack               |
| `drop`  | -          | -     | Discard the top item on the stack                 |
| `slide` | Number     | -     | Slide n items off the stack, keeping the top item |

### Arithmetic

Arithmetic instructions operate on the top two items on the stack and
replace them with the result of the operation. The first item pushed is
the left of the operator.

| Command | Parameters | Stack    | Meaning          |
| ------- | ---------- | -------- | ---------------- |
| `add`   | -          | LHS, RHS | Addition         |
| `sub`   | -          | LHS, RHS | Subtraction      |
| `mul`   | -          | LHS, RHS | Multiplication   |
| `div`   | -          | LHS, RHS | Integer Division |
| `mod`   | -          | LHS, RHS | Modulo           |

### Heap access

Heap access commands operate between the stack and heap. Values can be
stored in the heap for persistent addressed storage.

| Command    | Parameters | Stack           | Meaning                                  |
| ---------- | ---------- | --------------- | ---------------------------------------- |
| `store`    | -          | Address, Number | Store an item from the stack in the heap |
| `retrieve` | -          | Address         | Retrieve an item from the heap           |

### Control flow

Control flow consists of labels, unconditional and conditional jumps,
call, return, and exit. A control stack is used for call and return,
separate from the data stack.

| Command | Parameters | Stack  | Meaning                                                  |
| ------- | ---------- | ------ | -------------------------------------------------------- |
| `label` | Label      | -      | Mark a location in the program                           |
| `call`  | Label      | -      | Call a subroutine                                        |
| `jmp`   | Label      | -      | Jump unconditionally to a label                          |
| `jz`    | Label      | Number | Jump to a label if the top of the stack is zero          |
| `jn`    | Label      | Number | Jump to a label if the top of the stack is negative      |
| `ret`   | -          | -      | End a subroutine and transfer control back to the caller |
| `end`   | -          | -      | End the program                                          |

### I/O

I/O commands interact with the user for reading and writing numbers and
characters. The reference Haskell interpreter errors on EOF or
incorrectly formatted integers.

| Command  | Parameters | Meaning | Meaning                                            |
| -------- | ---------- | ------- | -------------------------------------------------- |
| `printc` | -          | Number  | Output the character at the top of the stack       |
| `printi` | -          | Number  | Output the number at the top of the stack          |
| `readc`  | -          | Address | Read a character and store it at the given address |
| `readi`  | -          | Address | Read a number and store it at the given address    |
