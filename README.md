![alt tag](https://raw.githubusercontent.com/krishicks/patrick/master/PatrickModelSeries.png)

Patrick can be used to pour a concrete struct from an interface declaration.

He takes as arguments the bytes of Go source code that includes an interface
declaration, the name of the interface within the source that you'd like to
pour a struct for, the name that should be given to the struct, and some
options.

He returns you `*ast.GenDecl`, which represents the struct, and
`[]*ast.FuncDecl`, which represents the methods defined on the struct. Each
method contains a simple body which should allow for the method to compile
successfully.

```Go
src := []byte(`
package mypackage

type MyInterface interface {
	A(int, string) (bool, error)
}`)

genDecl, funcDecls, err := patrick.Pour(src, "MyInterface", "myStruct")
if err != nil {
	panic(err)
}

decls := []ast.Decl{genDecl}
for _, funcDecl := range funcDecls {
	decls = append(decls, funcDecl)
}

f := &ast.File{
	Name:  ast.NewIdent("mypackage"),
	Decls: decls,
}

var buf bytes.Buffer
err = format.Node(&buf, token.NewFileSet(), f)
if err != nil {
	panic(err)
}

fmt.Println(buf.String())
```

The above example will output the following:

```
package mypackage

type myStruct struct {
}

func (m *myStruct) A(arg1 int, arg2 string) (bool, error) {
	var val1 bool
	var val2 error
	return val1, val2
}
```

Patrick does not know how to handle imports, but he doesn't need to. Use `imports.Process` to handle that:

```Go
src := []byte(`
package mypackage

import "net/http"

type MyInterface interface {
	A(http.Client, string)
}`)

genDecl, funcDecls, err := patrick.Pour(src, "MyInterface", "myStruct")
if err != nil {
	panic(err)
}

decls := []ast.Decl{genDecl}
for _, funcDecl := range funcDecls {
	decls = append(decls, funcDecl)
}

f := &ast.File{
	Name:  ast.NewIdent("mypackage"),
	Decls: decls,
}

var buf bytes.Buffer
err = format.Node(&buf, token.NewFileSet(), f)
if err != nil {
	panic(err)
}

out, err := imports.Process("src.go", buf.Bytes(), nil)
if err != nil {
	panic(err)
}

fmt.Println(string(out))
```

The above example will output the following:

```
package mypackage

import "net/http"

type myStruct struct {
}

func (m *myStruct) A(arg1 http.Client, arg2 string) {
}
```<Paste>
