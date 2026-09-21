# Classic McEliece Round 4 benchmarks

Benchmarks are centralized in `kem/classicmceliece/bench_test.go` rather than generated into each parameter-set package. The generated parameter-set packages contain implementation code only.

After changing generator templates, regenerate with:

```sh
go generate ./kem/classicmceliece
gofmt -w kem/classicmceliece
go test ./kem/classicmceliece/...
```

## Recommended benchmark commands

Run all high-level benchmarks from the consolidated benchmark file:

```sh
go test -run '^$' -bench . -benchmem ./kem/classicmceliece
```

Run representative operations for all parameter sets with one iteration each:

```sh
go test -run '^$' \
  -bench 'Benchmark(DeriveKeyPair|EncapsulateDeterministically|Decapsulate|UnmarshalPrivateKeyRecoverPublic)$' \
  -benchmem -benchtime=1x \
  ./kem/classicmceliece
```

Run one operation for one parameter set:

```sh
go test -run '^$' \
  -bench '^BenchmarkEncapsulateDeterministically/mceliece348864f$' \
  -benchmem -benchtime=10x \
  ./kem/classicmceliece
```

Generate a CPU profile for a high-level operation:

```sh
go test -run '^$' \
  -bench '^BenchmarkUnmarshalPrivateKeyRecoverPublic/mceliece8192128f$' \
  -benchmem -benchtime=3x \
  -cpuprofile mceliece.pprof \
  ./kem/classicmceliece
go tool pprof -top mceliece.pprof
```

## Benchmarks included

The consolidated benchmark file includes sub-benchmarks for all ten Round 4 parameter sets:

- `BenchmarkDeriveKeyPair/<variant>`
- `BenchmarkEncapsulateDeterministically/<variant>`
- `BenchmarkDecapsulate/<variant>`
- `BenchmarkMarshalPublicKey/<variant>`
- `BenchmarkUnmarshalPublicKey/<variant>`
- `BenchmarkMarshalPrivateKey/<variant>`
- `BenchmarkUnmarshalPrivateKeyRecoverPublic/<variant>`
- `BenchmarkPrivateKeyPublic/<variant>`

## Bottleneck summary from the scalar Go port

The current implementation is correctness-first and scalar. The dominant costs are:

1. Public-key generation and public-key recovery. `pkGen` and `pkGenFromSK` dominate key generation and private-key unmarshal. The cost comes mainly from building the generator matrix and running bit-matrix Gaussian elimination over `PKNRows * SysN/8` bytes. This is also why `UnmarshalBinaryPrivateKey` is expensive: the official compact secret key does not store the public key, so the wrapper reconstructs it.

2. Decapsulation. Decapsulation is much slower than encapsulation. The dominant internal work is `decrypt`, especially syndrome computation, root/evaluation work, and GF multiplication. It also runs `supportGen`, but support generation is smaller than the full decrypt path.

3. Encapsulation. Encapsulation is relatively cheap. The internal breakdown shows `syndrome` dominates `encrypt`; error-vector generation is minor in comparison.

4. Marshal/unmarshal copying. Public-key marshal, public-key unmarshal, and `PrivateKey.Public()` are linear copies of very large public keys. They are not algorithmic hot spots, but they allocate and copy hundreds of KiB to over 1 MiB per call.

5. Non-`f` key generation variance. Non-`f` variants can be much slower for some deterministic seeds because key generation retries until `pkGen` succeeds without the `f` pivot-column handling. The `f` variants are more stable for the seeds used in these benchmarks.

The most useful optimization targets are therefore:

- optimize `pkGen` / `pkGenFromSK` matrix elimination and row operations;
- avoid repeated public-key recovery where API usage permits storing or caching the public key;
- optimize syndrome computation in encapsulation and decapsulation;
- reduce allocations around SHAKE readers and output buffers;
- consider architecture-specific vectorized row operations behind build tags after the scalar code has been reviewed.
