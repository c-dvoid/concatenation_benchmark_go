# Go String Concatenation Benchmarks

I kept hearing "don't use `+` to build strings in Go, use `strings.Builder`", and I wanted to see for myself how much it actually matters and at what point. So I benchmarked six ways of joining strings, from 2 pieces up to 1000.

**Short version:** for a couple of strings, `+` is perfectly fine. Once you're joining in a loop, `+` gets slower *quadratically* — at 1000 pieces it was ~140× slower than a presized `strings.Builder` and allocated ~500× more memory.

## Approaches

| Function | Technique |
|---|---|
| `Plus` | `s += p` in a loop |
| `Sprintf` | `fmt.Sprintf("%s%s", s, p)` in a loop |
| `Builder` | `strings.Builder`, grows on demand |
| `BuilderGrow` | `strings.Builder` presized with `Grow` |
| `BytesBuffer` | `bytes.Buffer` |
| `Join` | `strings.Join(parts, "")` |

Every piece is the 5-byte string `"hello"`. All six functions are checked against `strings.Join` in `TestImplementations`, so the benchmarks compare implementations that produce identical output.

## Running

```bash
go test -run='^$' -bench=. -benchmem -count=10 > results.txt
benchstat results.txt
```

Install `benchstat` with `go install golang.org/x/perf/cmd/benchstat@latest`. My raw output is in [`results.txt`](results.txt).

> **PowerShell note:** Windows PowerShell splits `-bench=.` into `-bench=` and `.`,
> so no benchmarks run and you just get `[no tests to run]`. This one confused me for a while.
> Quote the flags instead: `go test "-run=^$" "-bench=." -benchmem`.

## Environment

- Go: `go1.26.1`
- OS / arch: `windows/amd64`
- CPU: AMD Ryzen 5 7535U (laptop)

## Results

Median of 10 runs from `benchstat`. The ± is the spread between runs.

### n = 2

| Approach | time/op | B/op | allocs/op |
|---|---|---|---|
| Plus | 34.0 ns ± 30% | 16 | 1 |
| Sprintf | 194.6 ns ± 58% | 64 | 5 |
| Builder | 44.8 ns ± 2% | 24 | 2 |
| BuilderGrow | 30.4 ns ± 49% | 16 | 1 |
| BytesBuffer | 74.2 ns ± 23% | 80 | 2 |
| Join | 49.6 ns ± 6% | 16 | 1 |

### n = 10

| Approach | time/op | B/op | allocs/op |
|---|---|---|---|
| Plus | 438.1 ns ± 5% | 328 | 9 |
| Sprintf | 1.70 µs ± 9% | 632 | 29 |
| Builder | 202.2 ns ± 20% | 120 | 4 |
| BuilderGrow | 104.3 ns ± 5% | 64 | 1 |
| BytesBuffer | 114.4 ns ± 42% | 128 | 2 |
| Join | 93.5 ns ± 8% | 64 | 1 |

### n = 100

| Approach | time/op | B/op | allocs/op |
|---|---|---|---|
| Plus | 9.46 µs ± 5% | 25.8 KiB | 99 |
| Sprintf | 20.77 µs ± 7% | 28.9 KiB | 299 |
| Builder | 920.7 ns ± 11% | 1016 | 7 |
| BuilderGrow | 601.1 ns ± 5% | 512 | 1 |
| BytesBuffer | 1.18 µs ± 26% | 1.4 KiB | 5 |
| Join | 870.4 ns ± 5% | 512 | 1 |

### n = 1000

| Approach | time/op | B/op | allocs/op |
|---|---|---|---|
| Plus | 880.7 µs ± 10% | 2.56 MiB | 999 |
| Sprintf | 1.84 ms ± 16% | 2.60 MiB | 3001 |
| Builder | 14.1 µs ± 7% | 17.5 KiB | 13 |
| BuilderGrow | 6.23 µs ± 5% | 5.25 KiB | 1 |
| BytesBuffer | 18.7 µs ± 10% | 21.2 KiB | 9 |
| Join | 8.25 µs ± 5% | 5.25 KiB | 1 |

The ± numbers at n=2 are large. These calls only take tens of nanoseconds, so a laptop's background activity and CPU boost clock changes add a lot of noise. I wouldn't read much into the small differences there. From n=10 up, the results are much more stable.

## What I learned

### `+` is fine for a few strings

At n=2, `Plus` is basically tied for first. `a + b` is one allocation of exactly the right size, and Go's runtime adds up the lengths before allocating, so there's nothing to optimize. The "never use `+`" advice is really about **loops**, not about `+` itself.

### `+` and `Sprintf` scale quadratically

Strings in Go are immutable, so `s += p` can't append in place. It allocates a new string and copies everything built so far into it. Building a 5 KB result out of 1000 pieces copies 5 + 10 + 15 + … + 5000 bytes, which is where the **2.56 MiB** allocated at n=1000 comes from. One allocation per piece also explains the 999 allocs/op.

You can see the quadratic growth in the timing: going from n=100 to n=1000 (10× more input) made `Plus` **93× slower**, while `BuilderGrow` only got about 10× slower, which is linear.

### Why `Sprintf` is the slowest

It does the same quadratic copying as `+`, and on top of that `fmt.Sprintf` has to parse the format string on every call and wrap its arguments in `any` interfaces. That's where the extra allocations come from: 3001 at n=1000 versus 999 for `Plus`. It's a nice tool for formatting, but not something to build strings with in a loop.

### Presizing matters: `Builder` vs `BuilderGrow`

`strings.Builder` stores a `[]byte` that it appends to, and when it runs out of room it allocates a bigger one (roughly double) and copies everything over. At n=1000 that happened 13 times. `BuilderGrow` adds up the lengths first and calls `Grow(n)`, so there is exactly **1 allocation** and every byte gets copied once. That made it more than 2× faster than plain `Builder` at n=1000 and cut the memory from 17.5 KiB to 5.25 KiB. Looping over the slice twice costs almost nothing compared with re-allocating.

### `bytes.Buffer` vs `strings.Builder`

`bytes.Buffer` grows its buffer the same way, but `buf.String()` has to **copy** the whole buffer into a new string, because the buffer could still be modified afterwards and a string must never change. `strings.Builder` is designed so `String()` can return its memory without copying. That extra copy shows up in the numbers: even at n=2, `BytesBuffer` allocated 80 bytes (a 64-byte starting buffer plus the 16-byte final string) to produce a 10-byte result.

### `BuilderGrow` vs `Join`

Both allocate exactly once, and they allocate the same number of bytes. I expected them to be identical, but `BuilderGrow` was 25–30% faster at n=100 and n=1000. Looking at the source of `strings.Join`, it:

- Checks for integer overflow on every length it adds up.
- Calls `WriteString` **twice** per element, once for the separator and once for the element, even when the separator is `""` like here.

So it's doing roughly double the `WriteString` calls. That's the price of being a general-purpose function that has to handle any separator safely. At n=10 `Join` actually came out slightly ahead. I'm not sure why yet. The gap is small, so it might just be noise.

## Benchmarking pitfall: the compiler can skip your work

My first version of this benchmark threw the result away:

```go
for b.Loop() {
    concatenation("hello", "world")
}
```

```
BenchmarkConcat-12    100000000    12.58 ns/op    0 B/op    0 allocs/op
```

A concatenation that creates a new string should allocate, but this showed **0 allocations**. The compiler's escape analysis could see that the result never left the loop, so the runtime built the 10-byte string in a small temporary buffer on the stack (Go uses a 32-byte one for exactly this case) instead of on the heap. Real code keeps its strings around, so this measured something that never happens in practice.

The fix is to store the result in a package-level variable, so the compiler has to assume it's used:

```go
var result string

for b.Loop() {
    result = concatenation(a, c)
}
```

```
BenchmarkConcat-12    45877171    26.13 ns/op    16 B/op    1 allocs/op
```

Now it allocates 16 bytes (10 bytes rounded up to the allocator's size class) and takes twice as long, which is the realistic number. My takeaway: if a benchmark looks suspiciously fast or shows 0 allocs, check whether the compiler is skipping the work.

## Conclusions

- For a handful of strings, just use `+`. It's readable and as fast as anything else.
- Never build a string with `+=` or `fmt.Sprintf` in a loop. The cost grows quadratically.
- For loops, use `strings.Builder`, and call `Grow` if you know (or can cheaply calculate) the final size.
- `strings.Join` is the easy choice when you already have a slice, and it's almost as fast as a hand-tuned builder.
- Prefer `strings.Builder` over `bytes.Buffer` when the end result is a string.
- Always check `allocs/op`, not just time. In this benchmark it explained almost every result.
