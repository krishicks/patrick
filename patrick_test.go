package patrick_test

import (
	"go/ast"
	"go/token"

	"github.com/krishicks/patrick"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Patrick", func() {
	Describe("Pour", func() {
		var (
			src  []byte
			opts patrick.Opts

			actualGenDecl   *ast.GenDecl
			actualFuncDecls []*ast.FuncDecl
			pourErr         error
		)

		BeforeEach(func() {
			src = []byte(`
package mypackage

type MyInterface interface {
	A()
	B()
}
`)
			opts = patrick.Opts{}
		})

		JustBeforeEach(func() {
			actualGenDecl, actualFuncDecls, pourErr = patrick.Pour(src, "MyInterface", "myStruct", opts)
			Expect(pourErr).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// fset := token.NewFileSet()

			// decls := []ast.Decl{actualGenDecl}
			// for _, fd := range actualFuncDecls {
			// 	decls = append(decls, fd)
			// }

			// f := &ast.File{
			// 	Name:  ast.NewIdent("mypackage"),
			// 	Decls: decls,
			// }

			// var buf bytes.Buffer
			// err := format.Node(&buf, fset, f)
			// if err != nil {
			// 	panic(err)
			// }

			// fmt.Println(buf.String())
		})

		It("returns a GenDecl with the correct type", func() {
			Expect(actualGenDecl.Tok).To(Equal(token.TYPE))
		})

		It("returns a GenDecl with a single spec", func() {
			// type myStruct struct {}

			Expect(actualGenDecl.Specs).To(HaveLen(1))

			spec, ok := actualGenDecl.Specs[0].(*ast.TypeSpec)
			Expect(ok).To(BeTrue())
			Expect(spec.Name.Name).To(Equal("myStruct"))
			Expect(spec.Name.Obj).To(Equal(ast.NewObj(ast.Typ, "myStruct")))
			expr, ok := spec.Type.(*ast.StructType)
			Expect(ok).To(BeTrue())
			Expect(expr).To(Equal(&ast.StructType{
				Fields: &ast.FieldList{},
			}))
		})

		It("returns a FuncDecl for each function in the interface", func() {
			// func(m *myStruct) A() {}
			// func(m *myStruct) B() {}

			Expect(actualFuncDecls).To(HaveLen(2))

			funcDecl := actualFuncDecls[0]
			Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))
			Expect(funcDecl.Type).To(Equal(&ast.FuncType{}))

			funcDecl = actualFuncDecls[1]
			Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "B"}))
			Expect(funcDecl.Type).To(Equal(&ast.FuncType{}))

			for _, funcDecl := range actualFuncDecls {
				receiver := funcDecl.Recv
				Expect(receiver.List).To(HaveLen(1))

				field := receiver.List[0]
				Expect(field.Names).To(HaveLen(1))

				name := field.Names[0]
				Expect(name.Name).To(Equal("m")) // TODO: this should be configurable, maybe by passing a func to generate it
				Expect(name.Obj).To(Equal(ast.NewObj(ast.Typ, "m")))

				expr, ok := field.Type.(*ast.StarExpr) // TODO: support non-pointer receiver
				Expect(ok).To(BeTrue())

				ident, ok := expr.X.(*ast.Ident)
				Expect(ok).To(BeTrue())

				Expect(ident).To(Equal(&ast.Ident{
					Name: "myStruct",
					Obj:  ast.NewObj(ast.Typ, "myStruct"),
				}))
			}
		})

		Context("when a function in the interface has params", func() {
			BeforeEach(func() {
				src = []byte(`
package mypackage

type MyInterface interface {
	A(int, string)
}
`)
			})

			It("returns a FuncDecl that takes the same params, assigning them generic names", func() {
				// func(m *myStruct) A(arg1 int, arg2 string) {}

				Expect(actualFuncDecls).To(HaveLen(1))

				funcDecl := actualFuncDecls[0]
				Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

				fields := funcDecl.Type.Params.List
				Expect(fields).To(HaveLen(2))

				Expect(fields[0].Names).To(HaveLen(1))
				Expect(fields[0].Names[0]).To(Equal(ast.NewIdent("arg1")))
				fieldType, ok := fields[0].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(fieldType.Name).To(Equal("int"))

				Expect(fields[1].Names).To(HaveLen(1))
				Expect(fields[1].Names[0]).To(Equal(ast.NewIdent("arg2")))
				fieldType, ok = fields[1].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(fieldType.Name).To(Equal("string"))
			})
		})

		Context("when a function in the interface has params with multiple names for the same type", func() {
			BeforeEach(func() {
				src = []byte(`
package mypackage

type MyInterface interface {
	A(someInt, anotherInt int)
}
`)
			})

			It("returns a FuncDecl that assigns them generic names and splits them into individual params", func() {
				// func(m *myStruct) A(arg1 int, arg2 int) {}

				Expect(actualFuncDecls).To(HaveLen(1))

				funcDecl := actualFuncDecls[0]
				Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

				fields := funcDecl.Type.Params.List
				Expect(fields).To(HaveLen(2))

				Expect(fields[0].Names).To(HaveLen(1))
				Expect(fields[0].Names[0]).To(Equal(ast.NewIdent("arg1")))
				fieldType, ok := fields[0].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(fieldType.Name).To(Equal("int"))

				Expect(fields[1].Names).To(HaveLen(1))
				Expect(fields[1].Names[0]).To(Equal(ast.NewIdent("arg2")))
				fieldType, ok = fields[1].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(fieldType.Name).To(Equal("int"))
			})
		})

		Context("when a function in the interface has return values", func() {
			BeforeEach(func() {
				src = []byte(`
package mypackage

type MyInterface interface {
	A() (int, string)
}
`)
			})

			It("returns a FuncDecl returning the same args", func() {
				// func(m *myStruct) A() (int, string) {
				//   var val1 int
				//   var val2 string
				//
				//   return val1, val2
				//	}

				Expect(actualFuncDecls).To(HaveLen(1))

				funcDecl := actualFuncDecls[0]
				Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

				fields := funcDecl.Type.Results.List
				Expect(fields).To(HaveLen(2))

				Expect(fields[0].Names).To(BeEmpty())
				fieldType, ok := fields[0].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(fieldType.Name).To(Equal("int"))

				Expect(fields[0].Names).To(BeEmpty())
				fieldType, ok = fields[1].Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(fieldType.Name).To(Equal("string"))

				// Body

				body := funcDecl.Body
				Expect(body.List).To(HaveLen(3))

				// var arg1 int
				declStmt, ok := body.List[0].(*ast.DeclStmt)
				Expect(ok).To(BeTrue())
				genDecl, ok := declStmt.Decl.(*ast.GenDecl)
				Expect(ok).To(BeTrue())

				Expect(genDecl.Tok).To(Equal(token.VAR))
				Expect(genDecl.Specs).To(HaveLen(1))
				spec, ok := genDecl.Specs[0].(*ast.ValueSpec)
				Expect(ok).To(BeTrue())
				Expect(spec.Names).To(HaveLen(1))

				ident := spec.Names[0]
				Expect(ident.Name).To(Equal("val1"))
				// not sure about including obj here; is it necessary?
				Expect(ident.Obj).NotTo(BeNil())
				Expect(ident.Obj.Kind).To(Equal(ast.Var))
				Expect(ident.Obj.Name).To(Equal("val1"))

				typ, ok := spec.Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(typ.Name).To(Equal("int"))

				// var arg2 string
				declStmt, ok = body.List[1].(*ast.DeclStmt)
				Expect(ok).To(BeTrue())
				genDecl, ok = declStmt.Decl.(*ast.GenDecl)
				Expect(ok).To(BeTrue())
				Expect(genDecl.Tok).To(Equal(token.VAR))

				Expect(genDecl.Tok).To(Equal(token.VAR))
				Expect(genDecl.Specs).To(HaveLen(1))
				spec, ok = genDecl.Specs[0].(*ast.ValueSpec)
				Expect(ok).To(BeTrue())
				Expect(spec.Names).To(HaveLen(1))

				ident = spec.Names[0]
				Expect(ident.Name).To(Equal("val2"))
				// not sure about including obj here; is it necessary?
				Expect(ident.Obj).NotTo(BeNil())
				Expect(ident.Obj.Kind).To(Equal(ast.Var))
				Expect(ident.Obj.Name).To(Equal("val2"))

				typ, ok = spec.Type.(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(typ.Name).To(Equal("string"))

				// return val1, val2
				returnStmt, ok := body.List[2].(*ast.ReturnStmt)
				Expect(ok).To(BeTrue())

				Expect(returnStmt.Results).To(HaveLen(2))
				result, ok := returnStmt.Results[0].(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(result.Name).To(Equal("val1"))

				result, ok = returnStmt.Results[1].(*ast.Ident)
				Expect(ok).To(BeTrue())
				Expect(result.Name).To(Equal("val2"))
			})

			Context("when the return values are named", func() {
				BeforeEach(func() {
					src = []byte(`
package mypackage

type MyInterface interface {
	A() (someInt int, someString string)
}
`)
				})

				It("returns a FuncDecl returning the same args with names stripped", func() {
					// func(m *myStruct) A() (int, string) {}

					Expect(actualFuncDecls).To(HaveLen(1))

					funcDecl := actualFuncDecls[0]
					Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

					fields := funcDecl.Type.Results.List
					Expect(fields).To(HaveLen(2))

					Expect(fields[0].Names).To(BeEmpty())
					fieldType, ok := fields[0].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("int"))

					Expect(fields[0].Names).To(BeEmpty())
					fieldType, ok = fields[1].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("string"))
				})
			})

			Context("when the return values have multiple names for a single type", func() {
				BeforeEach(func() {
					src = []byte(`
package mypackage

type MyInterface interface {
	A() (someInt, anotherInt int)
}
`)
				})

				It("returns a FuncDecl returning the same args with names stripped and individual return types", func() {
					// func(m *myStruct) A() (int, int) {}

					Expect(actualFuncDecls).To(HaveLen(1))

					funcDecl := actualFuncDecls[0]
					Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

					fields := funcDecl.Type.Results.List
					Expect(fields).To(HaveLen(2))

					Expect(fields[0].Names).To(BeEmpty())
					fieldType, ok := fields[0].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("int"))

					Expect(fields[0].Names).To(BeEmpty())
					fieldType, ok = fields[1].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("int"))
				})
			})
		})

		Context("when a function in the interface has named params", func() {
			BeforeEach(func() {
				src = []byte(`
package mypackage

type MyInterface interface {
	A(someInt int, someString string)
}
`)
			})

			Context("when preserveParamNames is true", func() {
				BeforeEach(func() {
					opts.PreserveParamNames = true
				})

				It("returns a FuncDecl with preserved param names", func() {
					// func(m *myStruct) A(someInt int, someString string) {}

					Expect(actualFuncDecls).To(HaveLen(1))

					funcDecl := actualFuncDecls[0]
					Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

					fields := funcDecl.Type.Params.List
					Expect(fields).To(HaveLen(2))

					Expect(fields[0].Names).To(HaveLen(1))
					Expect(fields[0].Names[0]).To(Equal(ast.NewIdent("someInt")))
					fieldType, ok := fields[0].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("int"))

					Expect(fields[1].Names).To(HaveLen(1))
					Expect(fields[1].Names[0]).To(Equal(ast.NewIdent("someString")))
					fieldType, ok = fields[1].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("string"))

				})
			})

			Context("when preserveParamNames is false", func() {
				BeforeEach(func() {
					opts.PreserveParamNames = false
				})

				It("returns a FuncDecl with generic param names", func() {
					// func(m *myStruct) A(someInt int, someString string) {}

					Expect(actualFuncDecls).To(HaveLen(1))

					funcDecl := actualFuncDecls[0]
					Expect(funcDecl.Name).To(Equal(&ast.Ident{Name: "A"}))

					fields := funcDecl.Type.Params.List
					Expect(fields).To(HaveLen(2))

					Expect(fields[0].Names).To(HaveLen(1))
					Expect(fields[0].Names[0]).To(Equal(ast.NewIdent("arg1")))
					fieldType, ok := fields[0].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("int"))

					Expect(fields[1].Names).To(HaveLen(1))
					Expect(fields[1].Names[0]).To(Equal(ast.NewIdent("arg2")))
					fieldType, ok = fields[1].Type.(*ast.Ident)
					Expect(ok).To(BeTrue())
					Expect(fieldType.Name).To(Equal("string"))
				})
			})
		})
	})
})
