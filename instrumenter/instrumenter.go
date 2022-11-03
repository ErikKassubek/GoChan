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
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

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
// TODO: spawn (als extra function), arguments
// TODO: select, receive in select must be <- a.getChan()
func instrument_ast(astSet *token.FileSet, f *ast.File) error {
	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		// ast.Print(astSet, n)

		switch n.(type) {

		case *ast.GenDecl: // add import of tracer lib if other libs get imported
			if n.(*ast.GenDecl).Tok == token.IMPORT {
				add_tracer_import(n)
			}
		case *ast.FuncDecl:
			if n.(*ast.FuncDecl).Name.Obj.Name == "main" {
				add_init_call(astSet, n)
			}
		case *ast.AssignStmt: // handle assign statements
			switch n.(*ast.AssignStmt).Rhs[0].(type) {
			case *ast.CallExpr: // call expression
				instrument_call_expressions(n)
			}
		case *ast.SendStmt: // handle send messages
			instrument_send_statement(n, c)
		case *ast.ExprStmt: // handle receive and close
			instrument_expression_statement(n, c)
		case *ast.GoStmt: // handle the creation of new go routines
			instrument_go_statements(astSet, n, c)
		case *ast.SelectStmt: // handel select statements
			instrument_select_statements(astSet, n, c)
		}

		return true
	})

	return nil
}

// add tracer lib import if other libs are imported
func add_tracer_import(n ast.Node) {
	specs := n.(*ast.GenDecl).Specs
	// add tracer lib to specs
	specs = append(specs, &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: "\"github.com/ErikKassubek/GoChan/tracer\"",
		},
	})
	n.(*ast.GenDecl).Specs = specs
}

func add_init_call(astSet *token.FileSet, n ast.Node) {
	body := n.(*ast.FuncDecl).Body.List
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
	n.(*ast.FuncDecl).Body.List = body
}

// instrument if n is a call expression
func instrument_call_expressions(n ast.Node) {
	// check make functions
	callExp := n.(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)

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
func instrument_send_statement(n ast.Node, c *astutil.Cursor) {
	// get the channel name
	channel := n.(*ast.SendStmt).Chan.(*ast.Ident).Name
	value := ""

	// get what is send through the channel
	v := n.(*ast.SendStmt).Value
	switch v.(type) {
	case (*ast.BasicLit):
		value = v.(*ast.BasicLit).Value
	case (*ast.Ident):
		value = v.(*ast.Ident).Name
	}

	// replace with function call
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
				0: &ast.Ident{
					Name: value,
				},
			},
		},
	})
}

// instrument receive and call statements
func instrument_expression_statement(n ast.Node, c *astutil.Cursor) {
	x_part := n.(*ast.ExprStmt).X
	switch x_part.(type) {
	case *ast.UnaryExpr:
		instrument_receive_statement(n, c)
	case *ast.CallExpr:
		instrument_close_statement(n, c)
	}
}

// instrument receive statements
func instrument_receive_statement(n ast.Node, c *astutil.Cursor) {
	x_part := n.(*ast.ExprStmt).X.(*ast.UnaryExpr)

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
func instrument_close_statement(n ast.Node, c *astutil.Cursor) {
	x_part := n.(*ast.ExprStmt).X.(*ast.CallExpr)

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
// TODO: not finished
func instrument_go_statements(astSet *token.FileSet, n ast.Node, c *astutil.Cursor) {
	fc := n.(*ast.GoStmt).Call.Fun
	switch function_call := fc.(type) {
	case *ast.FuncLit: // go with lambda
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
		// ast.Print(astSet, n)
	}

}

// instrument select statements
func instrument_select_statements(astSet *token.FileSet, n ast.Node, cur *astutil.Cursor) {
	// collect cases and replace <-i with i.GetChan()
	caseNodes := n.(*ast.SelectStmt).Body.List
	cases := make([]string, 0)
	d := false // check weather select contains default
	// ast.Print(astSet, n)
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
				n.(*ast.SelectStmt),
			},
		},
	)
}
