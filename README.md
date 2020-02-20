# gg - yet another generics workaround solution for go

Install:

```bash
go get github.com/zhiqiangxu/gg
```

Until Go2 finally comes, workaround for generics is necessary. **gg** can do what [`genny`](https://github.com/cheekybits/genny) can do in a cheaper way, no placeholder types like `generic.Type` is required; Besides that, **gg** can also accept multiple input files and merge them into one.


## Usage

```
gg -t TypeA=TypeB -import name=path -i inFile -o outFile
```

The above will replace `TypeA` with `TypeB`, and add a `name "path"` import spec to `inFile`, the resultant file is `outFile`.
`TypeA` can be any global type **defined** in `inFile`, `TypeB` can be simple identifier for some type or `name.type`, which `name` must be a valid reference to a package.


Multiple input files can be specified by multiple `-i`, in that case, they will first be merged into a single one.

## Real example

Given this code in `source.go`:

```go
package queue

// this is the target type to be replaced
type Something interface{}

// SomethingQueue is a queue.
type SomethingQueue struct {
  items []Something
}

func NewSomethingQueue() *SomethingQueue {
  return &SomethingQueue{items: make([]Something, 0)}
}
func (q *SomethingQueue) Push(item Something) {
  q.items = append(q.items, item)
}
func (q *SomethingQueue) Pop() Something {
  item := q.items[0]
  q.items = q.items[1:]
  return item
}
```

When `gg` is invoked like this:

```
gg -t Something=string -i source.go
```

It will replace all references to the global type `Something` with `string`, so the output is:

```go
package queue

// StringQueue is a queue.
type StringQueue struct {
  items []string
}

func NewStringQueue() *StringQueue {
  return &StringQueue{items: make([]string, 0)}
}
func (q *StringQueue) Push(item string) {
  q.items = append(q.items, item)
}
func (q *StringQueue) Pop() string {
  item := q.items[0]
  q.items = q.items[1:]
  return item
}
```

This means you can turn **any** existing implementation for some random type `TypeA` into one for type `TypeB`, **without** changing a word, enjoy it!
