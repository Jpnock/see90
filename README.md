# See90

See90 is a C90 compliant compiler which targets the MIPS I architecture.

It compiles single pre-processed files.

## Dependencies

- [Go](https://go.dev/dl/)
- [goyacc](https://pkg.go.dev/golang.org/x/tools/cmd/goyacc)
- [nex](https://github.com/blynn/nex)

These dependencies are automatically acquired via the `make` command.

## Building the compiler

To acquire dependencies and build the compiler, run the following in the root directory:

```bash
$ make bin/c_compiler
```

## Invoking the compiler

The compiler takes two flags

- `-S` for the input file path
- `-o` for the output file path

For example, it can be run as follows

```bash
$ ./bin/c_compiler -S "./test/all/main.c" -o "./test/all/main.s"
```

## Work-tracking

- The majority of work-tracking was done using [Monday](https://view.monday.com/2327051283-e57ce19b462981d12cde65d8d07e1882?r=use1)

## Credits

- [James Nock](https://github.com/Jpnock)
- [Dom Justice](https://github.com/DomJustice)
