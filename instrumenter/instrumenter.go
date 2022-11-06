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
traceElements.go
Type declarations for the trace elements
*/

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// collect params and there type
type arg_elem struct {
	name     string // variable name
	var_type string // variable type
	ellipsis bool   // ...type
}

// traverse all files for instrumentation
func instrument_files() error {
	for _, file := range file_names {
		err := instrument_file(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to instrument file %s.\n", file)
			return err
		}
	}
	return nil
}

// instrument file at given path and print to output
func instrument_file(file_path string) error {
	// create output file
	output_file, err := os.Create(out + file_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output file %s.\n", out+file_path)
		return err
	}
	defer output_file.Close()

	// copy mod and sum files
	if file_path[len(file_path)-3:] != ".go" {
		content, err := ioutil.ReadFile(file_path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file %s.\n", file_path)
			return err
		}
		_, err = output_file.Write(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to output file %s.\n", out+file_path)
			return err
		}
		return nil
	}

	// instrument go files
	err = instrument_go_file(file_path)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not instrument %s\n", in+file_path)
	}

	return nil
}

// instrument the given file in “in + file_path”
func instrument_go_file(file_path string) error {
	// get the ASP of the file
	astSet := token.NewFileSet()

	f, err := parser.ParseFile(astSet, file_path, nil, parser.AllErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse file %s\n", file_path)
		return err
	}

	instrument_ast(astSet, f)

	// print changed ast to output file
	output_file, err := os.OpenFile(out+file_path, os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open output file %s\n", out+file_path)
		return err
	}
	defer output_file.Close()

	if err := printer.Fprint(output_file, astSet, f); err != nil {
		return err
	}

	return nil

}

// instrument a given ast file f
func instrument_ast(astSet *token.FileSet, f *ast.File) error {
	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		switch n := n.(type) {
		case *ast.GenDecl: // add import of tracer lib if other libs get imported
			if n.Tok == token.IMPORT {
				add_tracer_import(n)
			}
		case *ast.FuncDecl:
			if n.Name.Obj != nil && n.Name.Obj.Name == "main" {
				add_init_call(n)
				if show_trace {
					add_show_trace_call(n)
				}
			} else {
				instrument_function_declarations(astSet, n, c)
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
		case *ast.GoStmt: // handle the creation of new go routines  // TODO: fix
			instrument_go_statements(astSet, n, c)
		case *ast.SelectStmt: // handel select statements  // TODO: fix, s. def
			instrument_select_statements(astSet, n, c)
		}

		return true
	})

	return nil
}

// add tracer lib import if other libs are imported
func add_tracer_import(n *ast.GenDecl) {
	specs := n.Specs
	// add tracer lib to specs
	specs = append(specs, &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: "\"github.com/ErikKassubek/GoChan/tracer\"",
		},
	})
	n.Specs = specs
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
	instrument_function_declaration_return_values(astSet, n)

	instrument_function_declaration_parameter(astSet, n)
}

// change the return value of functions if they contain a chan
func instrument_function_declaration_return_values(astSet *token.FileSet,
	n *ast.FuncDecl) {
	astResult := n.Type.Results

	// do nothing if the functions does not have return values
	if astResult == nil {
		return
	}

	// traverse all return types
	for i, res := range n.Type.Results.List {
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
		n.Type.Results.List[i] = &ast.Field{
			Type: &ast.Ident{
				Name: translated_string,
			},
		}
	}
}

// instrument all function parameter, replace them by args_rand any... and
// add declarations and casting in body
func instrument_function_declaration_parameter(astSet *token.FileSet, n *ast.FuncDecl) {
	type parameter struct {
		name     string
		val_type string
		ellipse  bool
	}

	// collect parameters
	param_list := n.Type.Params.List
	if param_list == nil { // function has no parameter
		return
	}

	if n.Body == nil || n.Body.List == nil { // function is empty
		return
	}

	parameter_list := make([]parameter, 0)

	// traverse parameters
	for _, param := range param_list {
		name := param.Names[0].Name

		val_type := ""
		ellipse := false

		switch t := param.Type.(type) {

		case *ast.Ident:
			val_type = t.Name

		case *ast.StructType:
			val_type = "struct{}"

		case *ast.ArrayType:
			switch t_elt := t.Elt.(type) {
			case *ast.Ident:
				val_type = "[]" + t_elt.Name
			case *ast.StarExpr:
				val_type = "[]*" + t_elt.X.(*ast.Ident).Name
			}

		case *ast.StarExpr:
			switch t_x := t.X.(type) {
			case *ast.Ident:
				val_type = "*" + t_x.Name
			case *ast.SelectorExpr:
				val_type = "*" + get_selector_expression_name(t_x)
			}

		case *ast.SelectorExpr:
			val_type = get_selector_expression_name(t)

		case *ast.ChanType:
			switch t_type := t.Value.(type) {
			case *ast.Ident:
				val_type = "chan " + t_type.Name
			case *ast.StructType:
				val_type = "chan struct{}"
			}
		case *ast.InterfaceType:
			val_type = "interface{}"
		case *ast.FuncType:
			val_type = "func("
			if t.Params.List != nil { // function parameter
				for i, elem := range t.Params.List {
					switch elem_type := elem.Type.(type) {
					case *ast.Ident:
						val_type += elem_type.Name
					case *ast.SelectorExpr:
						val_type += get_selector_expression_name(elem.Type.(*ast.SelectorExpr))
					}
					val_type += " "
					switch t_type_type := elem.Type.(type) {
					case *ast.Ident:
						val_type += t_type_type.Name
					case *ast.StarExpr:
						val_type += ("*" + t_type_type.X.(*ast.Ident).Name)
					}
					if i != len(t.Params.List)-1 {
						val_type += ", "
					}
				}
			}

		case *ast.MapType:
			val_type = "map["
			switch t_key := t.Key.(type) {
			case *ast.Ident:
				val_type += t_key.Name
			case *ast.StarExpr:
				val_type += "*" + t_key.X.(*ast.Ident).Name
			}
			val_type += "]"
			switch t_val := t.Value.(type) {
			case *ast.Ident:
				val_type += t_val.Name
			case *ast.StarExpr:
				val_type += "*" + t_val.X.(*ast.Ident).Name
			}
		case *ast.Ellipsis:
			ellipse = true
			switch t_elt := t.Elt.(type) {
			case *ast.Ident:
				val_type += t_elt.Name
			case *ast.InterfaceType:
				val_type += "interface{}"
			case *ast.StarExpr:
				val_type += ("*" + t_elt.X.(*ast.Ident).Name)

			}
		}

		parameter_list = append(parameter_list, parameter{name: name, val_type: val_type, ellipse: ellipse})
	}

	// replace parameter list with arg_rand ...any
	paramName := "args_" + randSeq(10)
	n.Type.Params.List = []*ast.Field{
		&ast.Field{
			Names: []*ast.Ident{
				&ast.Ident{
					Name: paramName,
				},
			},
			Type: &ast.Ident{
				Name: "...any",
			},
		},
	}

	// add declarations of parameters in function body
	declarations := make([]ast.Stmt, 0)
	for i, elem := range parameter_list {
		if elem.ellipse {
			declarations = append(declarations, &ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.Ident{
						Name: elem.name,
					},
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.Ident{
						Name: "make([]" + elem.val_type + ",0)",
					},
				},
			})
			declarations = append(declarations, &ast.RangeStmt{
				Key: &ast.Ident{
					Name: "_, arg_loop_var",
					Obj: &ast.Object{
						Kind: ast.Var,
						Name: "_",
						Decl: &ast.AssignStmt{
							Lhs: []ast.Expr{
								&ast.Ident{
									Name: "_",
									Obj: &ast.Object{
										Kind: ast.Var,
										Name: "_",
									},
								},
								&ast.Ident{
									Name: "arg_loop_var",
									Obj: &ast.Object{
										Kind: ast.Var,
										Name: "arg_loop_var",
									},
								},
							},
							Tok: token.DEFINE,
							Rhs: []ast.Expr{
								&ast.UnaryExpr{
									Op: token.RANGE,
									X: &ast.SliceExpr{
										X: &ast.Ident{
											Name: paramName,
										},
										Low: &ast.BasicLit{
											Kind:  token.INT,
											Value: strconv.Itoa(i),
										},
										Slice3: false,
									},
								},
							},
						},
					},
				},
				Tok: token.DEFINE,
				X: &ast.SliceExpr{
					X: &ast.Ident{
						Name: paramName,
					},
					Low: &ast.BasicLit{
						Kind:  token.INT,
						Value: strconv.Itoa(i),
					},
					Slice3: false,
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{
								&ast.Ident{
									Name: elem.name,
								},
							},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.Ident{
										Name: "append",
									},
									Args: []ast.Expr{
										&ast.Ident{
											Name: elem.name,
										},
										&ast.TypeAssertExpr{
											X: &ast.Ident{
												Name: "arg_loop_var",
											},
											Type: &ast.Ident{
												Name: elem.val_type,
											},
										},
									},
								},
							},
						},
					},
				},
			})
		} else {
			rhsName := ""
			rhsName = paramName + "[" + strconv.Itoa(i) + "].(" + elem.val_type + ")"

			declarations = append(declarations, &ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.Ident{
						Name: elem.name,
					},
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.Ident{
						Name: rhsName,
					},
				},
			})
		}

	}

	n.Body.List = append(declarations, n.Body.List...)
}

func instrument_receive_with_assign(astSet *token.FileSet, n *ast.AssignStmt, c *astutil.Cursor) {
	if n.Rhs[0].(*ast.UnaryExpr).Op != token.ARROW {
		return
	}

	variable := n.Lhs[0].(*ast.Ident).Name
	var channel string

	switch x := n.Rhs[0].(*ast.UnaryExpr).X.(type) {
	case *ast.Ident:
		channel = x.Name
	case *ast.SelectorExpr:
		channel = get_selector_expression_name(x)
	}
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

	if callExp.Fun.(*ast.Ident).Name == "make" {
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
	var channel string
	switch c := n.Chan.(type) {
	case *ast.Ident:
		channel = c.Name
	case *ast.SelectorExpr:
		channel = get_selector_expression_name(c)
	}

	value := ""

	// get what is send through the channel
	v := n.Value
	// fmt.Printf("%T\n", v)
	call_expr := false
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
			value = "[]" + lit_type.Elt.(*ast.Ident).Name + "{" + lit.Elts[0].(*ast.Ident).Name + "}"
		}
	case *ast.SelectorExpr:
		value = get_selector_expression_name(lit)
	case *ast.UnaryExpr:
		value = lit.Op.String() + lit.X.(*ast.CompositeLit).Type.(*ast.Ident).Name + "{}"
	default:
		errString := fmt.Sprintf("Unknown type %T in instrument_send_statement", v)
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
	default:
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
		switch exp := x_part_x.Fun.(type) {
		case *ast.SelectorExpr:
			sel_exp := x_part_x.Fun.(*ast.SelectorExpr)
			channel = get_selector_expression_name(sel_exp)
		case *ast.Ident:
			channel = exp.Name
		}
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
	if x_part.Fun.(*ast.Ident).Name != "close" {
		return
	}

	var channel string
	switch name_elem := x_part.Args[0].(type) {
	case *ast.Ident:
		channel = name_elem.Name
	case *ast.SelectorExpr:
		channel = get_selector_expression_name(name_elem)
	default:
		errString := fmt.Sprintf("Unknown type %T in instrument_send_statement", x_part.Args[0])
		panic(errString)
	}

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
	fc := n.Call.Fun
	switch function_call := fc.(type) {
	case *ast.FuncLit: // go with lambda
		// collect arguments
		arguments := []arg_elem{}
		for _, arg := range function_call.Type.Params.List {
			ellipsis := false
			type_val := ""
			switch t := arg.Type.(type) {
			case *ast.Ident:
				type_val = t.Name
			case *ast.Ellipsis:
				type_val = t.Elt.(*ast.Ident).Name
				ellipsis = true
			}

			arguments = append(arguments, arg_elem{arg.Names[0].Name, type_val, ellipsis})
		}

		arg_name := "args_" + randSeq(10)
		function_call.Type.Params = &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{
						{
							Name: arg_name,
						},
					},
					Type: &ast.Ellipsis{
						Elt: &ast.Ident{
							Name: "any",
						},
					},
				},
			},
		}

		// add argument assignment
		// add variable declarations
		var decl []ast.Stmt
		for i, param := range arguments {
			if param.ellipsis {
				argument_creation := []ast.Stmt{
					&ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{
										{
											Name: param.name,
											Obj: &ast.Object{
												Kind: ast.Var,
												Name: param.name,
											},
										},
									},
									Type: &ast.ArrayType{
										Elt: &ast.Ident{
											Name: param.var_type,
										},
									},
								},
							},
						},
					},
					&ast.RangeStmt{
						Key: &ast.Ident{
							Name: "_, arg_loop_var",
							Obj: &ast.Object{
								Kind: ast.Var,
								Name: "_",
								Decl: &ast.AssignStmt{
									Lhs: []ast.Expr{
										&ast.Ident{
											Name: "_",
											Obj: &ast.Object{
												Kind: ast.Var,
												Name: "_",
											},
										},
										&ast.Ident{
											Name: "arg_loop_var",
											Obj: &ast.Object{
												Kind: ast.Var,
												Name: "arg_loop_var",
											},
										},
									},
									Tok: token.DEFINE,
									Rhs: []ast.Expr{
										&ast.UnaryExpr{
											Op: token.RANGE,
											X: &ast.SliceExpr{
												X: &ast.Ident{
													Name: arg_name,
												},
												Low: &ast.BasicLit{
													Kind:  token.INT,
													Value: strconv.Itoa(i),
												},
												Slice3: false,
											},
										},
									},
								},
							},
						},
						Tok: token.DEFINE,
						X: &ast.SliceExpr{
							X: &ast.Ident{
								Name: arg_name,
							},
							Low: &ast.BasicLit{
								Kind:  token.INT,
								Value: strconv.Itoa(i),
							},
							Slice3: false,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.AssignStmt{
									Lhs: []ast.Expr{
										&ast.Ident{
											Name: param.name,
										},
									},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{
										&ast.CallExpr{
											Fun: &ast.Ident{
												Name: "append",
											},
											Args: []ast.Expr{
												&ast.Ident{
													Name: param.name,
												},
												&ast.TypeAssertExpr{
													X: &ast.Ident{
														Name: "arg_loop_var",
													},
													Type: &ast.Ident{
														Name: param.var_type,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}
				decl = append(decl, argument_creation...)

			} else {
				decl = append(decl, &ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{
									{
										Name: param.name,
									},
								},
								Type: &ast.Ident{
									Name: param.var_type,
								},
								Values: []ast.Expr{
									&ast.TypeAssertExpr{
										X: &ast.IndexExpr{
											X: &ast.Ident{
												Name: arg_name,
											},
											Index: &ast.BasicLit{
												Kind:  token.INT,
												Value: strconv.Itoa(i),
											},
										},
										Type: &ast.Ident{
											Name: param.var_type,
										},
									},
								},
							},
						},
					},
				})
			}
		}
		function_call.Body.List = append(decl, function_call.Body.List...)

		// arguments of function call
		func_arg := append([]ast.Expr{function_call}, n.Call.Args...)

		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.Ident{
					Name: "tracer.Spawn",
				},
				Args: func_arg,
			},
		})
	case *ast.Ident, *ast.SelectorExpr: // go with function
		var name string
		switch function_type_inner := fc.(type) {
		case *ast.Ident:
			name = function_type_inner.Name
		case *ast.SelectorExpr:
			name = get_selector_expression_name(function_type_inner)
		}

		func_args := []ast.Expr{&ast.Ident{
			Name: name,
		}}

		func_args = append(func_args, n.Call.Args...)

		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "tracer",
					},
					Sel: &ast.Ident{
						Name: "Spawn",
					},
				},
				Args: func_args,
			},
		})
	case *ast.CallExpr: // TODO: finish blocking/kubernetes58107
		var name string
		switch fun := fc.(*ast.CallExpr).Fun.(type) {
		case *ast.Ident:
			name = fun.Name
		case *ast.SelectorExpr:
			name = get_selector_expression_name(fun)
		}

		// fmt.Println(name)

		// ast.Print(astSet, fc)

		args := fc.(*ast.CallExpr).Args
		arg_val := []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.Ident{
					Name: name,
				},
				Args: args,
			},
		}

		arg_val = append(arg_val, n.Call.Args...)

		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "tracer",
					},
					Sel: &ast.Ident{
						Name: "Spawn",
					},
				},
				Args: arg_val,
			},
		})
	default:
		errString := fmt.Sprintf("Unknown type %T in instrument_go_statement", fc)
		panic(errString)
	}

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
			c.(*ast.CommClause).Body = append(c.(*ast.CommClause).Body, &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.Ident{
						Name: "tracer.PostDefault",
					},
				},
			})
			continue
		}

		// TODO: add brackets with arguments after function call
		switch c_type := c.(*ast.CommClause).Comm.(type) {
		case *ast.ExprStmt: // receive in switch without assign
			f := c_type.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr)

			// check for receive
			if f.Sel.Name != "Receive" {
				continue
			}
			name := f.X.(*ast.Ident).Name
			cases = append(cases, name)

			f.X.(*ast.Ident).Name = "<-" + name
			f.Sel.Name = "GetChan"

		case *ast.AssignStmt: // receive with assign
			f := c_type.Rhs[0]
			name := strings.Split(f.(*ast.CallExpr).Fun.(*ast.Ident).Name, ".")
			c.(*ast.CommClause).Comm.(*ast.AssignStmt).Rhs[0] = &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "<-" + name[0],
				},
				Sel: &ast.Ident{
					Name: "GetChan()",
				},
			}
		}
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

// get a random sequence of letters
func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
