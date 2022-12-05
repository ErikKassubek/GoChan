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
instrumentMutex.go
Instrument mutex in files
*/

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// instrument a given ast file f
func instrument_mutex(f *ast.File) error {
	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		switch n_type := n.(type) {
		case *ast.FuncDecl:
			instrument_function_declarations_mut(n_type, c)
		case *ast.DeclStmt:
			instrument_mutex_decl(n_type, c)
		case *ast.GenDecl: // add import of goChan lib if other libs get imported
			instrument_gen_decl_mut(n_type, c)
		case *ast.AssignStmt:
			switch n_type.Rhs[0].(type) {
			case *ast.CompositeLit:
				instrument_assign_struct_mut(n_type, c)
			}
		}

		return true
	})
	return nil
}

func instrument_function_declarations_mut(d *ast.FuncDecl, c *astutil.Cursor) {
	instrument_function_declaration_return_values_mut(d.Type)
	instrument_function_declaration_parameter_mut(d.Type)
}

func instrument_mutex_decl(d *ast.DeclStmt, c *astutil.Cursor) {
	switch d.Decl.(type) {
	case *ast.GenDecl:
	default: // not a sync.Mutex
		return
	}

	var n *ast.ValueSpec
	switch spec_type := d.Decl.(*ast.GenDecl).Specs[0].(type) {
	case *ast.ValueSpec:
		n = spec_type
	}

	mutexType := ""
	goChanTypePointer := false
	name := ""
	var x_val *ast.SelectorExpr

	if n == nil || n.Type == nil {
		return
	}

	switch n.Type.(type) {
	case *ast.SelectorExpr:
		x_val = n.Type.(*ast.SelectorExpr)
	case *ast.StarExpr:
		switch n.Type.(*ast.StarExpr).X.(type) {
		case *ast.SelectorExpr:
			goChanTypePointer = true
			x_val = n.Type.(*ast.StarExpr).X.(*ast.SelectorExpr)
		default:
			return
		}
	default: // not a sync.Mutex
		return
	}

	switch x_type := x_val.X.(type) {
	case *ast.Ident:
		if x_type.Name != "sync" { // not a sync.Mutex
			return
		}
	default: // not a sync.Mutex
		return
	}

	if x_val.Sel.Name == "Mutex" {
		mutexType = "NewMutex"
	} else if x_val.Sel.Name == "RWMutex" {
		mutexType = "NewRWMutex"
	} else { // not a sync.Mutex
		return
	}
	name = n.Names[0].Name
	varTyp := "goChan." + mutexType + "()"
	if goChanTypePointer {
		varTyp = "goChan." + mutexType + "(); " + name + ":= &" + name + "_"
		name += "_"
	}

	c.Replace(&ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.Ident{
				Name: name,
				Obj: &ast.Object{
					Kind: ast.ObjKind(token.VAR),
					Name: name,
				},
			},
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.Ident{
				Name: varTyp,
			},
		},
	})
}

// instrument mutex in struct declaration
func instrument_gen_decl_mut(n *ast.GenDecl, c *astutil.Cursor) {
	for j, s := range n.Specs {
		switch s_type := s.(type) {
		case *ast.ValueSpec: // var
			genString := ""
			name := get_name(s_type.Type)
			if name == "sync.Mutex" {
				genString = "= goChan.NewMutex()"
			} else if name == "sync.RWMutex" {
				genString = "= goChan.NewRWMutex()"
			} else {
				continue
			}
			n.Specs[j].(*ast.ValueSpec).Type = &ast.Ident{Name: genString}

		case *ast.TypeSpec: // struct or interface
			switch s_t_t := s_type.Type.(type) {
			case *ast.StructType: // struct
				for i, field := range s_t_t.Fields.List {
					name_str := ""
					if name := get_name(field.Type); name == "sync.Mutex" {
						name_str = "Mutex"
					} else if name == "sync.RWMutex" {
						name_str = "RWMutex"
					} else {
						continue
					}

					n.Specs[j].(*ast.TypeSpec).Type.(*ast.StructType).Fields.List[i].Type.(*ast.SelectorExpr).X.(*ast.Ident).Name = "goChan"
					n.Specs[j].(*ast.TypeSpec).Type.(*ast.StructType).Fields.List[i].Type.(*ast.SelectorExpr).Sel.Name = name_str
				}
			case *ast.InterfaceType:
				for _, t := range s_t_t.Methods.List {
					switch t_type := t.Type.(type) {
					case *ast.FuncType:
						instrument_function_declaration_return_values_mut(t_type)
						instrument_function_declaration_parameter_mut(t_type)
					}
				}
			}
		}
	}
}

// change the return value of functions if they contain a mutex
func instrument_function_declaration_return_values_mut(n *ast.FuncType) {
	astResult := n.Results

	// do nothing if the functions does not have return values
	if astResult == nil {
		return
	}

	// traverse all return types
	var mut_name string
	for i, res := range n.Results.List {
		switch res.Type.(type) {
		case *ast.SelectorExpr:
			if name := get_name(res.Type); name == "sync.Mutex" {
				mut_name = "goChan.Mutex"
			} else if name == "sync.RWMutex" {
				mut_name = "goChan.RWMutex"
			} else {
				continue
			}
		case *ast.StarExpr:
			if name := get_name(res.Type.(*ast.StarExpr).X); name == "sync.Mutex" {
				mut_name = "*goChan.Mutex"
			} else if name == "sync.RWMutex" {
				mut_name = "*goChan.RWMutex"
			} else {
				continue
			}
		default:
			continue // continue if not a channel
		}

		// set the translated value
		n.Results.List[i] = &ast.Field{
			Type: &ast.Ident{
				Name: mut_name,
			},
		}
	}
}

// instrument all function parameter
func instrument_function_declaration_parameter_mut(n *ast.FuncType) {
	astResult := n.Params

	// do nothing if the functions does not have return values
	if astResult == nil {
		return
	}

	// traverse all parameters
	var mut_name string
	for i, res := range astResult.List {
		switch res.Type.(type) {
		case *ast.SelectorExpr:
			if name := get_name(res.Type); name == "sync.Mutex" {
				mut_name = "goChan.Mutex"
			} else if name == "sync.RWMutex" {
				mut_name = "goChan.RWMutex"
			} else {
				continue
			}
		case *ast.StarExpr:
			if name := get_name(res.Type.(*ast.StarExpr).X); name == "sync.Mutex" {
				mut_name = "*goChan.Mutex"
			} else if name == "sync.RWMutex" {
				mut_name = "*goChan.RWMutex"
			} else {
				continue
			}
		default:
			continue // continue if not a channel
		}

		// set the translated value
		n.Params.List[i] = &ast.Field{
			Names: n.Params.List[i].Names,
			Type: &ast.Ident{
				Name: mut_name,
			},
		}
	}
}

// instrument mutex in assign of struct type
func instrument_assign_struct_mut(n *ast.AssignStmt, c *astutil.Cursor) {
	for i, t := range n.Rhs[0].(*ast.CompositeLit).Elts {
		switch t.(type) {
		case *(ast.KeyValueExpr):
		default:
			continue
		}

		switch t_type := t.(*ast.KeyValueExpr).Value.(type) {
		case *ast.CompositeLit:
			var name string
			if get_name(t_type.Type) == "sync.Mutex" {
				name = "*goChan.NewMutex()"
			} else if get_name(t_type.Type) == "sync.RWMutex" {
				name = "*goChan.NewRWMutex()"
			} else {
				continue
			}

			n.Rhs[0].(*ast.CompositeLit).Elts[i].(*ast.KeyValueExpr).Value = &ast.Ident{
				Name: name,
			}
		}

	}

	var name_str string
	if n.Rhs[0].(*ast.CompositeLit).Elts == nil {
		if name := get_name(n.Rhs[0].(*ast.CompositeLit).Type); name == "sync.Mutex" {
			name_str = "goChan.NewMutex()"
		} else if name == "sync.RWMutex" {
			name_str = "goChan.NewRWMutex()"
		} else {
			return
		}

		n.Rhs[0] = &ast.Ident{
			Name: name_str,
		}
	}
}
