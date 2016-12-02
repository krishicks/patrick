package patrick

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// Opts are the options to use when pouring the struct
type Opts struct {
	// Whether to preserve parameter varNames from the interface declaration
	PreserveParamNames bool
}

// Pour takes bytes of a Go source file, an interface in the source, and
// options for how to pour the struct
func Pour(src []byte, interfaceName string, structName string, opts Opts) (*ast.GenDecl, []*ast.FuncDecl, error) {
	if interfaceName == "" {
		return nil, nil, errors.New("must provide interface varName")
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, parser.AllErrors)
	if err != nil {
		return nil, nil, err
	}

	var obj *ast.Object
	ast.Inspect(f, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		switch v := node.(type) {
		case *ast.TypeSpec:
			if v.Name != nil && v.Name.Name != interfaceName {
				return true
			}

			obj = v.Name.Obj
			return false
		default:
			return true
		}
	})

	if obj == nil {
		return nil, nil, errors.New("could not find interface")
	}

	// structObj not required when not using pointer receiver
	structObj := ast.NewObj(ast.Typ, structName)
	recvObj := ast.NewObj(ast.Typ, string(structName[0]))

	genDecl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{
					Name: structObj.Name,
					Obj:  structObj,
				},
				Type: &ast.StructType{
					Fields: &ast.FieldList{
						List: nil,
					},
				},
			},
		},
	}

	funcDecls := getFuncDecls(structObj, recvObj, obj, []ast.Decl{}, opts.PreserveParamNames)

	return genDecl, funcDecls, nil
}

func getFuncDecls(
	structObj, recvObj, o *ast.Object,
	funcDecls []ast.Decl,
	preserveParamNames bool,
) []*ast.FuncDecl {
	newFuncDecls := []*ast.FuncDecl{}

	typeSpec, ok := o.Decl.(*ast.TypeSpec)
	if !ok {
		panic("not ok")
	}

	typ, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		panic("not ok")
	}

	for _, field := range typ.Methods.List {
		switch v := field.Type.(type) {
		case *ast.FuncType:
			newFuncDecls = append(newFuncDecls, newFuncDecl(
				recvObj,
				structObj,
				field.Names[0].Name,
				v.Params,
				v.Results,
				preserveParamNames,
			))
		case *ast.Ident:
			if v.Obj == nil {
				panic("found embedded interface with no associated interface")
			}

			newFuncDecls = append(newFuncDecls, getFuncDecls(structObj, recvObj, v.Obj, funcDecls, preserveParamNames)...)
		default:
			fmt.Printf("%#v\n", v)
			continue
		}
	}

	return newFuncDecls
}

func newFuncDecl(
	recvObj *ast.Object,
	structObj *ast.Object,
	funcName string,
	params *ast.FieldList,
	results *ast.FieldList,
	preserveParamNames bool,
) *ast.FuncDecl {
	funcDecl := &ast.FuncDecl{
		Name: ast.NewIdent(funcName),
		Type: &ast.FuncType{},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{
							Name: recvObj.Name,
							Obj:  recvObj,
						},
					},
					// TODO: support non-pointer receiver
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: structObj.Name,
							Obj:  structObj,
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			Lbrace: token.NoPos,
			List:   nil,
			Rbrace: token.NoPos,
		},
	}

	if params.NumFields() > 0 {
		var fl = new(ast.FieldList)
		var i int
		for _, field := range params.List {
			if len(field.Names) > 0 {
				for _, ident := range field.Names {
					i++

					var varName string
					if preserveParamNames {
						varName = ident.Name
					} else {
						varName = fmt.Sprintf("arg%d", i)
					}

					fl.List = append(fl.List, &ast.Field{
						Names: []*ast.Ident{ast.NewIdent(varName)},
						Type:  field.Type,
					})
				}
			} else {
				i++
				fl.List = append(fl.List, &ast.Field{
					Names: []*ast.Ident{ast.NewIdent(fmt.Sprintf("arg%d", i))},
					Type:  field.Type,
				})
			}
		}

		funcDecl.Type.Params = fl
	}

	if results.NumFields() > 0 {
		var stmts []ast.Stmt
		var returnStmt = new(ast.ReturnStmt)
		var fl = new(ast.FieldList)
		var i int
		var recordField = func(field *ast.Field) {
			i++
			fl.List = append(fl.List, &ast.Field{
				Type: field.Type,
			})

			varName := fmt.Sprintf("val%d", i)

			varType, ok := field.Type.(*ast.Ident)
			if !ok {
				panic(fmt.Sprintf("could not get ident for field: %#v", field))
			}

			stmts = append(stmts, &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{
								{
									Name: varName,
									Obj:  ast.NewObj(ast.Var, varName),
								},
							},
							Type: ast.NewIdent(varType.Name),
						},
					},
				},
			})

			returnStmt.Results = append(returnStmt.Results, ast.NewIdent(varName))
		}

		for _, field := range results.List {
			recordField(field)
			for i := 1; i < len(field.Names); i++ {
				recordField(field)
			}
		}

		funcDecl.Type.Results = fl
		funcDecl.Body.List = append(stmts, returnStmt)
	}

	return funcDecl
}
