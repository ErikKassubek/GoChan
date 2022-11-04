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

/*
	TODO: arguments in go lambda
	TODO: handle chan type (extra todo)
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
	ast.Print(astSet, f)
	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		switch n := n.(type) {

		case *ast.GenDecl: // add import of tracer lib if other libs get imported
			if n.Tok == token.IMPORT {
				add_tracer_import(n)
			}
		case *ast.FuncDecl:
			if n.Name.Obj.Name == "main" {
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
				instrument_call_expressions(n)
			}
		case *ast.SendStmt: // handle send messages
			instrument_send_statement(astSet, n, c)
		case *ast.ExprStmt: // handle receive and close
			instrument_expression_statement(n, c)
		case *ast.GoStmt: // handle the creation of new go routines
			instrument_go_statements(astSet, n, c)
		case *ast.SelectStmt: // handel select statements
			instrument_select_statements(n, c)
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
	param_objects := n.Type.Params.List

	// ignore functions without params
	if param_objects == nil {
		param_objects = []*ast.Field{}
	}

	params := make([]arg_elem, 0)

	for _, param := range param_objects {
		name := param.Names[0].Name
		var var_type string
		var ellipsis bool
		// TODO: handle chan type
		switch elem := param.Type.(type) {
		case *ast.Ident:
			var_type = elem.Name
			ellipsis = false
		case *ast.Ellipsis:
			var_type = elem.Elt.(*ast.Ident).Name
			ellipsis = true
		default:
			panic("Unknown type in instrument_function_declarations")
		}

		params = append(params, arg_elem{name, var_type, ellipsis})
	}

	// replace function arguments with args ...any_randomString
	arg_name := "args_" + randSeq(10)
	arg_statement := []*ast.Field{
		{
			Names: []*ast.Ident{
				{Name: arg_name},
			},
			Type: &ast.Ellipsis{
				Elt: &ast.Ident{
					Name: "any",
				},
			},
		},
	}

	n.Type.Params.List = arg_statement

	// add variable declarations
	var decl []ast.Stmt
	for i, param := range params {
		if param.ellipsis { // list
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
	n.Body.List = append(decl, n.Body.List...)
}

// instrument if n is a call expression
func instrument_call_expressions(n *ast.AssignStmt) {
	// check make functions
	callExp := n.Rhs[0].(*ast.CallExpr)

	// don't change call expression of non-make function
	switch callExp.Fun.(type) {
	case *ast.IndexExpr:
		return
	}

	if callExp.Fun.(*ast.Ident).Name == "make" {
		switch callExp.Args[0].(type) {
		// make creates a channel
		case *ast.ChanType:
			// get type of channel
			chanType := callExp.Args[0].(*ast.ChanType).Value.(*ast.Ident).Name

			// sey size of channel
			chanSize := "0"
			if len(callExp.Args) >= 2 {
				chanSize = callExp.Args[1].(*ast.BasicLit).Value
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
func instrument_send_statement(astStd *token.FileSet, n *ast.SendStmt, c *astutil.Cursor) {
	// get the channel name
	channel := n.Chan.(*ast.Ident).Name
	value := ""

	// get what is send through the channel
	v := n.Value
	call_expr := false
	switch lit := v.(type) {
	case (*ast.BasicLit):
		value = lit.Value
	case (*ast.Ident):
		value = lit.Name
	case (*ast.CallExpr):
		call_expr = true
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
func instrument_expression_statement(n *ast.ExprStmt, c *astutil.Cursor) {
	x_part := n.X
	switch x_part.(type) {
	case *ast.UnaryExpr:
		instrument_receive_statement(n, c)
	case *ast.CallExpr:
		instrument_close_statement(n, c)
	}
}

// instrument receive statements
func instrument_receive_statement(n *ast.ExprStmt, c *astutil.Cursor) {
	x_part := n.X.(*ast.UnaryExpr)

	// check if correct operation
	if x_part.Op != token.ARROW {
		return
	}

	// get channel name
	channel := x_part.X.(*ast.Ident).Name

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
func instrument_close_statement(n *ast.ExprStmt, c *astutil.Cursor) {
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

	channel := x_part.Args[0].(*ast.Ident).Name
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
// TODO: go statements with lambda functions with arguments
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

		c.Replace(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.Ident{
					Name: "tracer.Spawn",
				},
				Args: []ast.Expr{
					function_call,
				},
			},
		})
	case *ast.Ident:
		name := function_call.Name

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
	}

}

// instrument select statements
func instrument_select_statements(n *ast.SelectStmt, cur *astutil.Cursor) {
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

		f := c.(*ast.CommClause).Comm.(*ast.ExprStmt).X.(*ast.CallExpr).Fun.(*ast.SelectorExpr)
		// check for receive
		if f.Sel.Name != "Receive" {
			continue
		}
		name := f.X.(*ast.Ident).Name
		cases = append(cases, name)

		f.X.(*ast.Ident).Name = "<-" + name
		f.Sel.Name = "GetChan"
	}

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

// get a random sequence of letters
func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
