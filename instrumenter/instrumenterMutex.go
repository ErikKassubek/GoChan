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
func instrument_mutex(astSet *token.FileSet, f *ast.File) error {
	astutil.Apply(f, nil, func(c *astutil.Cursor) bool {
		n := c.Node()

		switch n_type := n.(type) {
		case *ast.DeclStmt:
			instrument_mutex_decl(n_type, c)
		}

		return true
	})
	return nil
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
	switch n.Type.(type) {
	case *ast.SelectorExpr:
	default: // not a sync.Mutex
		return
	}

	switch x_type := n.Type.(*ast.SelectorExpr).X.(type) {
	case *ast.Ident:
		if x_type.Name != "sync" { // not a sync.Mutex
			return
		}
	default: // not a sync.Mutex
		return
	}

	if n.Type.(*ast.SelectorExpr).Sel.Name == "Mutex" {
		mutexType = "NewLock"
	} else if n.Type.(*ast.SelectorExpr).Sel.Name == "RWMutex" {
		mutexType = "NewRWLock"
	} else { // not a sync.Mutex
		return
	}

	name := n.Names[0].Name

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
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "tracer",
					},
					Sel: &ast.Ident{
						Name: mutexType,
					},
				},
			},
		},
	})
}
