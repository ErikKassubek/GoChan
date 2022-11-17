package main

/*
Copyright (c) 2022, Erik Kassubek
All rights reserved.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

/*
Author: Erik Kassubek <erik-kassubek@t-online.de>
Package: GoChan-Instrumenter
Project: Bachelor Thesis at the Albert-Ludwigs-University Freiburg,
	Institute of Computer Science: Dynamic Analysis of message passing go programs
*/

/*
instrumenter.go
instrument files to work with the "github.com/ErikKassubek/GoChan/tracer" library
*/

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// instrument a given ast file f
func instrument_chan(astSet *token.FileSet, f *ast.File) error {
	add_tracer_import(f)

	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Name.Obj != nil && n.Name.Obj.Name == "main" {
				add_init_call(n)
				if show_trace {
					add_show_trace_call(n)
				}
			} else {
				instrument_function_declarations(astSet, n, c)
			}
		}
		return true
	})

	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		switch n := n.(type) {
		case *ast.GenDecl: // add import of tracer lib if other libs get imported
			if n.Tok == token.IMPORT {

			}
		case *ast.AssignStmt: // handle assign statements
			switch n.Rhs[0].(type) {
			case *ast.CallExpr: // call expression
				instrument_call_expressions(astSet, n)
			case *ast.UnaryExpr: // receive with assign
				instrument_receive_with_assign(astSet, n, c)
			}
		case *ast.SendStmt: // handle send messages
			instrument_send_statement(astSet, n, c)
		case *ast.ExprStmt: // handle receive and close
			instrument_expression_statement(astSet, n, c)
		case *ast.GoStmt: // handle the creation of new go routines
			instrument_go_statements(astSet, n, c)
		case *ast.SelectStmt: // handel select statements
			instrument_select_statements(astSet, n, c)
		}

		return true
	})

	return nil
}

// add tracer lib import
func add_tracer_import(n *ast.File) {
	import_spec := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: "\"github.com/ErikKassubek/GoChan/tracer\"",
		},
	}

	if n.Decls == nil {
		n.Decls = []ast.Decl{}
	}

	if len(n.Decls) == 0 {
		n.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT}}, n.Decls...)
	}

	switch n.Decls[0].(type) {
	case (*ast.GenDecl):
		if n.Decls[0].(*ast.GenDecl).Tok != token.IMPORT {
			n.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT}}, n.Decls...)
		}
	default:
		n.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT}}, n.Decls...)
	}

	switch n.Decls[0].(type) {
	case *ast.GenDecl:
	default:
		n.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT}}, n.Decls...)
	}

	if n.Decls[0].(*ast.GenDecl).Specs == nil {
		n.Decls[0].(*ast.GenDecl).Specs = []ast.Spec{}
	}

	n.Decls[0].(*ast.GenDecl).Specs = append(n.Decls[0].(*ast.GenDecl).Specs, import_spec)
}

func add_init_call(n *ast.FuncDecl) {
	body := n.Body.List
	if body == nil {
		return
	}

	body = append([]ast.Stmt{
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.Ident{
					Name: "tracer.Init",
				},
			},
		},
	}, body...)
	n.Body.List = body
}

// add function to show the trace
func add_show_trace_call(n *ast.FuncDecl) {
	n.Body.List = append(n.Body.List, &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.Ident{
				Name: "tracer.PrintTrace",
			},
		},
	})
}

func instrument_function_declarations(astSet *token.FileSet, n *ast.FuncDecl, c *astutil.Cursor) {
	instrument_function_declaration_return_values(astSet, n.Type)

	instrument_function_declaration_parameter(astSet, n.Type)
}

// change the return value of functions if they contain a chan
func instrument_function_declaration_return_values(astSet *token.FileSet,
	n *ast.FuncType) {
	astResult := n.Results

	// do nothing if the functions does not have return values
	if astResult == nil {
		return
	}

	// traverse all return types
	for i, res := range n.Results.List {
		switch res.Type.(type) {
		case *ast.ChanType: // do not call continue if channel
		default:
			continue // continue if not a channel
		}

		translated_string := ""
		switch v := res.Type.(*ast.ChanType).Value.(type) {
		case *ast.Ident: // chan <type>
			translated_string = "tracer.Chan[" + v.Name + "]"
		case *ast.StructType:
			translated_string = "tracer.Chan[struct{}]"
		case *ast.ArrayType:
			translated_string = "tracer.Chan[[]" + v.Elt.(*ast.Ident).Name + "]"
		}

		// set the translated value
		n.Results.List[i] = &ast.Field{
			Type: &ast.Ident{
				Name: translated_string,
			},
		}
	}
}

// instrument all function parameter
func instrument_function_declaration_parameter(astSet *token.FileSet, n *ast.FuncType) {
	astResult := n.Params

	// do nothing if the functions does not have return values
	if astResult == nil {
		return
	}

	// traverse all parameters
	for i, res := range astResult.List {
		switch res.Type.(type) {
		case *ast.ChanType: // do not call continue if channel
		default:
			continue // continue if not a channel
		}

		translated_string := ""
		switch v := res.Type.(*ast.ChanType).Value.(type) {
		case *ast.Ident: // chan <type>
			translated_string = "tracer.Chan[" + v.Name + "]"
		case *ast.StructType:
			translated_string = "tracer.Chan[struct{}]"
		case *ast.ArrayType:
			translated_string = "tracer.Chan[[]" + v.Elt.(*ast.Ident).Name + "]"
		}

		// set the translated value
		n.Params.List[i] = &ast.Field{
			Names: n.Params.List[i].Names,
			Type: &ast.Ident{
				Name: translated_string,
			},
		}
	}
}

func instrument_receive_with_assign(astSet *token.FileSet, n *ast.AssignStmt, c *astutil.Cursor) {
	if n.Rhs[0].(*ast.UnaryExpr).Op != token.ARROW {
		return
	}

	variable := n.Lhs[0].(*ast.Ident).Name
	channel := get_name(astSet, n.Rhs[0].(*ast.UnaryExpr).X)

	token := n.Tok
	c.Replace(&ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: variable,
			},
		},
		Tok: token,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.Ident{
					Name: channel + ".Receive",
				},
			},
		},
	})
}

// instrument if n is a call expression
func instrument_call_expressions(astSet *token.FileSet, n *ast.AssignStmt) {
	// check make functions
	callExp := n.Rhs[0].(*ast.CallExpr)

	// don't change call expression of non-make function
	switch callExp.Fun.(type) {
	case *ast.IndexExpr, *ast.SelectorExpr:
		return
	}

	if get_name(astSet, callExp.Fun) == "make" {
		switch callExp.Args[0].(type) {
		// make creates a channel
		case *ast.ChanType:
			// get type of channel

			var chanType string
			callExpVal := callExp.Args[0].(*ast.ChanType).Value
			switch val := callExpVal.(type) {
			case *ast.Ident:
				chanType = val.Name
			case *ast.StructType:
				var struct_elem string
				for i, elem := range val.Fields.List {
					struct_elem += elem.Names[0].Name + " " + elem.Type.(*ast.Ident).Name
					if i == len(val.Fields.List)-1 {
						struct_elem += ", "
					}
				}
				chanType = "struct{" + struct_elem + "}"
			}

			// set size of channel
			chanSize := "0"
			if len(callExp.Args) >= 2 {
				switch args_type := callExp.Args[1].(type) {
				case *ast.BasicLit:
					chanSize = args_type.Value
				case *ast.Ident:
					chanSize = args_type.Name

				}
			}

			// set function name to tracer.NewChan[<chanType>]
			callExp.Fun.(*ast.Ident).Name = "tracer.NewChan[" + chanType + "]"

			// remove second argument if size was given in make
			if len(callExp.Args) >= 1 {
				callExp.Args = callExp.Args[:1]
			}

			// set function argument to channel size
			callExp.Args[0] = &ast.BasicLit{Kind: token.INT, Value: chanSize}
		}
	}
}

// instrument a send statement
func instrument_send_statement(astSet *token.FileSet, n *ast.SendStmt, c *astutil.Cursor) {
	// get the channel name
	channel := get_name(astSet, n.Chan)

	value := ""

	// get what is send through the channel
	v := n.Value
	// fmt.Printf("%T\n", v)
	call_expr := false
	func_lit := false
	switch lit := v.(type) {
	case (*ast.BasicLit):
		value = lit.Value
	case (*ast.Ident):
		value = lit.Name
	case (*ast.CallExpr):
		call_expr = true
	case *ast.ParenExpr:
		value = n.Chan.(*ast.Ident).Obj.Decl.(*ast.Field).Type.(*ast.ChanType).Value.(*ast.Ident).Name
	case *ast.CompositeLit:
		switch lit_type := lit.Type.(type) {
		case *ast.StructType:
			value = "switch{}{}"
		case *ast.ArrayType:
			value = "[]" + get_name(astSet, lit_type.Elt) + "{" + get_name(astSet, lit.Elts[0]) + "}"
		}
	case *ast.SelectorExpr:
		value = get_selector_expression_name(lit)
	case *ast.UnaryExpr:
		switch t_type := lit.X.(*ast.CompositeLit).Type.(type) {
		case *ast.Ident:
			value = lit.Op.String() + t_type.Name + "{}"
		case *ast.SelectorExpr:
			value = lit.Op.String() + get_selector_expression_name(t_type) + "{}"

		}
	case *ast.FuncLit:
		func_lit = true
	case *ast.IndexExpr:
		value = get_name(astSet, lit.X) + "[" + get_name(astSet, lit.Index)
	default:
		ast.Print(astSet, v)
		errString := fmt.Sprintf("Unknown type %T in instrument_send_statement2", v)
		panic(errString)
	}

	// replace with function call
	if call_expr {
		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: channel,
					},
					Sel: &ast.Ident{
						Name: "Send",
					},
				},
				Lparen: token.NoPos,
				Args: []ast.Expr{
					v.(*ast.CallExpr),
				},
			},
		})
	} else if func_lit {
		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: channel,
					},
					Sel: &ast.Ident{
						Name: "Send",
					},
				},
				Lparen: token.NoPos,
				Args: []ast.Expr{
					v.(*ast.FuncLit),
				},
			},
		})
	} else {
		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: channel,
					},
					Sel: &ast.Ident{
						Name: "Send",
					},
				},
				Lparen: token.NoPos,
				Args: []ast.Expr{
					&ast.Ident{
						Name: value,
					},
				},
			},
		})
	}
}

// instrument receive and call statements
func instrument_expression_statement(astSet *token.FileSet, n *ast.ExprStmt, c *astutil.Cursor) {
	x_part := n.X
	switch x_part.(type) {
	case *ast.UnaryExpr:
		instrument_receive_statement(astSet, n, c)
	case *ast.CallExpr:
		instrument_close_statement(astSet, n, c)
	case *ast.TypeAssertExpr:
	default:
		ast.Print(astSet, x_part)
		errString := fmt.Sprintf("Unknown type %T in instrument_expression_statement", x_part)
		panic(errString)
	}
}

// instrument receive statements
func instrument_receive_statement(astSet *token.FileSet, n *ast.ExprStmt, c *astutil.Cursor) {
	x_part := n.X.(*ast.UnaryExpr)

	// check if correct operation
	if x_part.Op != token.ARROW {
		return
	}

	// get channel name
	var channel string
	switch x_part_x := x_part.X.(type) {
	case *ast.Ident:
		channel = x_part_x.Name
	case *ast.CallExpr:
		channel = get_name(astSet, x_part_x.Fun)
	case *ast.SelectorExpr:
		channel = get_selector_expression_name(x_part_x)
	default:
		errString := fmt.Sprintf("Unknown type %T in instrument_receive_statement", x_part)
		panic(errString)
	}

	c.Replace(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: channel,
				},
				Sel: &ast.Ident{
					Name: "Receive",
				},
			},
		},
	})
}

// change close statements to tracer.Close
func instrument_close_statement(astSet *token.FileSet, n *ast.ExprStmt, c *astutil.Cursor) {
	x_part := n.X.(*ast.CallExpr)

	// return if not ident
	wrong := true
	switch x_part.Fun.(type) {
	case *ast.Ident:
		wrong = false
	}
	if wrong {
		return
	}

	// remove all non close statements
	if x_part.Fun.(*ast.Ident).Name != "close" || len(x_part.Args) == 0 {
		return
	}

	channel := get_name(astSet, x_part.Args[0])
	c.Replace(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: channel,
				},
				Sel: &ast.Ident{
					Name: "Close",
				},
			},
		},
	})
}

// instrument the creation of new go routines
func instrument_go_statements(astSet *token.FileSet, n *ast.GoStmt, c *astutil.Cursor) {
	var_name := "GoChanRoutineIndex"

	var func_body *ast.BlockStmt
	switch t := n.Call.Fun.(type) {
	case *ast.FuncLit:
		func_body = &ast.BlockStmt{
			List: t.Body.List,
		}
	case *ast.Ident, *ast.SelectorExpr:
		func_body = &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun:  n.Call.Fun,
						Args: n.Call.Args,
					},
				},
			},
		}
	default:
		fmt.Printf("Unknown Type %T in instrument_go_statement", n.Call.Fun)
	}

	n = &ast.GoStmt{
		Call: &ast.CallExpr{
			Fun:  n.Call.Fun,
			Args: n.Call.Args,
		},
	}

	fc := n.Call.Fun
	switch fc_type := fc.(type) {
	case *ast.FuncLit: // go with lambda
		instrument_function_declaration_return_values(astSet, fc_type.Type)
		instrument_function_declaration_parameter(astSet, fc_type.Type)
	default:
		n.Call.Args = nil
	}

	params := &ast.FieldList{}
	switch n.Call.Fun.(type) {
	case *ast.FuncLit:
		params = n.Call.Fun.(*ast.FuncLit).Type.Params
	}

	// add PreSpawn
	c.Replace(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{
								&ast.Ident{
									Name: var_name,
								},
							},
							Tok: token.DEFINE,
							Rhs: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.Ident{
										Name: "tracer.SpawnPre",
									},
								},
							},
						},
						&ast.GoStmt{
							Call: &ast.CallExpr{
								Fun: &ast.FuncLit{
									Type: &ast.FuncType{
										Params: params,
									},
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											&ast.ExprStmt{
												X: &ast.CallExpr{
													Fun: &ast.Ident{
														Name: "tracer.SpawnPost",
													},
													Args: []ast.Expr{
														&ast.Ident{
															Name: var_name,
														},
													},
												},
											},
											func_body,
										},
									},
								},
								Args: n.Call.Args,
							},
						},
					},
				},
			},
		},
	})

}

// instrument select statements
func instrument_select_statements(astSet *token.FileSet, n *ast.SelectStmt, cur *astutil.Cursor) {
	// collect cases and replace <-i with i.GetChan()
	caseNodes := n.Body.List
	cases := make([]string, 0)
	d := false // check weather select contains default
	for _, c := range caseNodes {
		// only look at communication cases
		switch c.(type) {
		case *ast.CommClause:
		default:
			continue
		}

		// check for default, add tracer.PostDefault if found
		if c.(*ast.CommClause).Comm == nil {
			d = true
			c.(*ast.CommClause).Body = append([]ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.Ident{
							Name: "tracer.PostDefault",
						},
					},
				}}, c.(*ast.CommClause).Body...)
			continue
		}

		var name string
		switch c_type := c.(*ast.CommClause).Comm.(type) {
		case *ast.ExprStmt: // receive in switch without assign
			f := c_type.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr)

			// check for receive
			if f.Sel.Name != "Receive" {
				continue
			}
			name = f.X.(*ast.Ident).Name
			cases = append(cases, name)

			f.X.(*ast.Ident).Name = "<-" + name
			f.Sel.Name = "GetChan"

		case *ast.AssignStmt: // receive with assign (:=)
			assign_name := get_name(astSet, c_type.Lhs[0])
			f := c_type.Rhs[0]
			name = strings.Split(f.(*ast.CallExpr).Fun.(*ast.Ident).Name, ".")[0]

			c.(*ast.CommClause).Comm.(*ast.AssignStmt).Lhs[0] = &ast.Ident{
				Name: assign_name + "_sel",
			}
			c.(*ast.CommClause).Comm.(*ast.AssignStmt).Rhs[0] = &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "<-" + name,
				},
				Sel: &ast.Ident{
					Name: "GetChan()",
				},
			}
			c.(*ast.CommClause).Body = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{
							Name: assign_name,
						},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.SelectorExpr{
							X: &ast.Ident{
								Name: assign_name + "_sel",
							},
							Sel: &ast.Ident{
								Name: "GetInfo()",
							},
						},
					},
				},
			}, c.(*ast.CommClause).Body...)
		}

		// add post select
		c.(*ast.CommClause).Body = append([]ast.Stmt{
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: name,
						},
						Sel: &ast.Ident{
							Name: "PostSelect",
						},
					},
				},
			},
		}, c.(*ast.CommClause).Body...)
	}

	// add cases to preSelect
	cases_string := "false, "
	if d {
		cases_string = "true, "
	}
	for i, c := range cases {
		cases_string += (c + ".GetId()")
		if i != len(cases)-1 {
			cases_string += ", "
		}
	}

	// add tracer.PreSelect
	cur.Replace(
		&ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.Ident{
							Name: "tracer.PreSelect",
						},
						Args: []ast.Expr{
							&ast.Ident{
								Name: cases_string,
							},
						},
					},
				},
				n,
			},
		},
	)
}

// get name
func get_name(astSet *token.FileSet, n ast.Expr) string {
	switch n_type := n.(type) {
	case *ast.Ident:
		return n_type.Name
	case *ast.SelectorExpr:
		return get_selector_expression_name(n_type)
	case *ast.StarExpr:
		return "*" + get_name(astSet, n.(*ast.StarExpr).X)
	case *ast.CallExpr:
		arguments := make([]string, 0)
		for _, a := range n.(*ast.CallExpr).Args {
			arguments = append(arguments, get_name(astSet, a))
		}
		name := get_name(astSet, n.(*ast.CallExpr).Fun) + "("
		for _, a := range arguments {
			name += a + ", "
		}
		name += ")"
		return name
	case *ast.ParenExpr:
		return get_name(astSet, n_type.X)
	case *ast.TypeAssertExpr:
		return get_name(astSet, n_type.Type)
	case *ast.FuncType:
		name := "func("
		if n_type.Params != nil {
			for _, a := range n_type.Params.List {
				name += get_name(astSet, a.Type) + ", "
			}
		}
		name += ")"
		if n_type.Results != nil {
			name += "("
			for _, a := range n_type.Results.List {
				name += get_name(astSet, a.Type) + ", "
			}
			name += ")"
		}
		return name
	case *ast.FuncLit:
		return get_name(astSet, n_type.Type)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ArrayType:
		return "[]" + get_name(astSet, n_type.Elt)
	case *ast.BasicLit:
		return n_type.Value
	case *ast.ChanType:
		return "chan " + get_name(astSet, n_type.Value)
	case *ast.StructType:
		return "struct{}"
	case *ast.IndexExpr:
		return get_name(astSet, n_type.X) + "[" + get_name(astSet, n_type.Index) + "]"
	case *ast.BinaryExpr:
		return get_name(astSet, n_type.X) + n_type.Op.String() + get_name(astSet, n_type.Y)
	case *ast.UnaryExpr:
		return n_type.Op.String() + get_name(astSet, n_type.X)
	case *ast.MapType:
		return "map[" + get_name(astSet, n_type.Key) + "]" + get_name(astSet, n_type.Value)
	case *ast.Ellipsis:
		return "..." + get_name(astSet, n_type.Elt)
	default:
		ast.Print(astSet, n)
		panic(fmt.Sprintf("Could not get name of type %T\n", n))
	}
}

// get the full name of an selector expression
func get_selector_expression_name(n *ast.SelectorExpr) string {
	switch x := n.X.(type) {
	case *ast.Ident:
		return x.Name + "." + n.Sel.Name
	case *ast.SelectorExpr:
		return get_selector_expression_name(x) + "." + n.Sel.Name
	}
	return ""
}
