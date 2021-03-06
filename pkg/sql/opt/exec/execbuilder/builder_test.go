// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package execbuilder

// This file is home to the execbuild tests, which are similar to the logic
// tests.
//
// Each testfile contains testcases of the form
//   <command> [<args>]...
//   <SQL statement or expression>
//   ----
//   <expected results>
//
// The supported commands are:
//
//  - exec-raw
//
//    Runs a SQL statement against the database (not through the execbuilder).
//
//  - build
//
//    Builds a memo structure from a SQL query and outputs a representation of
//    the "expression view" of the memo structure. Note: tests for the build
//    process belong in the optbuilder tests; this is here only to have the
//    expression view in the testfiles (for documentation).
//
//  - exec
//
//    Builds a memo structure from a SQL statement, then builds an
//    execution plan and runs it, outputting the results.
//
//  - exec-explain
//
//    Builds a memo structure from a SQL statement, then builds an
//    execution plan and outputs the details of that plan.
//
//  - catalog
//
//    Prints information about a table, retrieved through the Catalog interface.
//
// The supported args are:
//
//  - vars=(type1,type2,...)
//
//    Information about IndexedVar columns.
//
//  - allow-unsupported
//
//    Allows building unsupported scalar expressions into UnsupportedExprOp.

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/settings/cluster"
	"github.com/cockroachdb/cockroach/pkg/sql/opt"
	"github.com/cockroachdb/cockroach/pkg/sql/opt/exec"
	"github.com/cockroachdb/cockroach/pkg/sql/opt/optbuilder"
	"github.com/cockroachdb/cockroach/pkg/sql/opt/xform"
	"github.com/cockroachdb/cockroach/pkg/sql/parser"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
	"github.com/cockroachdb/cockroach/pkg/testutils/datadriven"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/treeprinter"
)

var (
	testDataGlob = flag.String("d", "testdata/[^.]*", "test data glob")
)

func TestBuild(t *testing.T) {
	defer leaktest.AfterTest(t)()

	paths, err := filepath.Glob(*testDataGlob)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no testfiles found matching: %s", *testDataGlob)
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			ctx := context.Background()
			semaCtx := tree.MakeSemaContext(false /* privileged */)
			evalCtx := tree.MakeTestingEvalContext(cluster.MakeTestingClusterSettings())

			s, sqlDB, _ := serverutils.StartServer(t, base.TestServerArgs{})
			defer s.Stopper().Stop(ctx)

			_, err := sqlDB.Exec("CREATE DATABASE test; SET DATABASE = test;")
			if err != nil {
				t.Fatal(err)
			}

			datadriven.RunTest(t, path, func(d *datadriven.TestData) string {
				var allowUnsupportedExpr, rowSort bool

				for _, arg := range d.CmdArgs {
					switch arg.Key {
					case "allow-unsupported":
						allowUnsupportedExpr = true

					case "rowsort":
						// We will sort the resulting rows before comparing with the
						// expected result.
						rowSort = true

					default:
						d.Fatalf(t, "unknown argument: %s", arg.Key)
					}
				}

				switch d.Cmd {
				case "exec-raw":
					_, err := sqlDB.Exec(d.Input)
					if err != nil {
						d.Fatalf(t, "%v", err)
					}
					return ""

				case "build", "exec", "exec-explain":
					// Parse the SQL.
					stmt, err := parser.ParseOne(d.Input)
					if err != nil {
						d.Fatalf(t, "%v", err)
					}

					eng := s.Executor().(exec.TestEngineFactory).NewTestEngine("test")
					defer eng.Close()

					// Build and optimize the opt expression tree.
					o := xform.NewOptimizer(&evalCtx, xform.OptimizeAll)
					builder := optbuilder.New(
						ctx, &semaCtx, &evalCtx, eng.Catalog(), o.Factory(), stmt,
					)
					builder.AllowUnsupportedExpr = allowUnsupportedExpr
					root, props, err := builder.Build()
					if err != nil {
						d.Fatalf(t, "BuildOpt: %v", err)
					}
					ev := o.Optimize(root, props)

					if d.Cmd == "build" {
						return ev.String()
					}

					// Build the execution node tree.
					node, err := New(eng.Factory(), ev).Build()
					if err != nil {
						d.Fatalf(t, "BuildExec: %v", err)
					}

					var columns sqlbase.ResultColumns

					// Execute the node tree.
					var results []tree.Datums
					if d.Cmd == "exec-explain" {
						results, err = eng.Explain(node)
					} else {
						columns = eng.Columns(node)
						results, err = eng.Execute(node)
					}
					if err != nil {
						d.Fatalf(t, "Exec: %v", err)
					}

					if rowSort {
						sort.Slice(results, func(i, j int) bool {
							for k := range results[i] {
								cmp := results[i][k].Compare(&evalCtx, results[j][k])
								if cmp != 0 {
									return cmp < 0
								}
							}
							return false
						})
					}

					// Format the results.
					var buf bytes.Buffer
					tw := tabwriter.NewWriter(
						&buf,
						2,   /* minwidth */
						1,   /* tabwidth */
						2,   /* padding */
						' ', /* padchar */
						0,   /* flags */
					)
					if columns != nil {
						for i := range columns {
							if i > 0 {
								fmt.Fprintf(tw, "\t")
							}
							fmt.Fprintf(tw, "%s:%s", columns[i].Name, columns[i].Typ)
						}
						fmt.Fprintf(tw, "\n")
					}
					for _, r := range results {
						for j, val := range r {
							if j > 0 {
								fmt.Fprintf(tw, "\t")
							}
							if d, ok := val.(*tree.DString); ok && utf8.ValidString(string(*d)) {
								str := string(*d)
								if str == "" {
									str = "·"
								}
								// Avoid the quotes on strings.
								fmt.Fprintf(tw, "%s", str)
							} else {
								fmt.Fprintf(tw, "%s", val)
							}
						}
						fmt.Fprintf(tw, "\n")
					}
					_ = tw.Flush()
					return buf.String()

				case "catalog":
					// Create the engine in order to get access to its catalog.
					eng := s.Executor().(exec.TestEngineFactory).NewTestEngine("test")
					defer eng.Close()

					parts := strings.Split(d.Input, ".")
					name := tree.NewTableName(tree.Name(parts[0]), tree.Name(parts[1]))
					tbl, err := eng.Catalog().FindTable(context.Background(), name)
					if err != nil {
						d.Fatalf(t, "Catalog: %v", err)
					}

					tp := treeprinter.New()
					opt.FormatCatalogTable(tbl, tp)
					return tp.String()

				default:
					d.Fatalf(t, "unsupported command: %s", d.Cmd)
					return ""
				}
			})
		})
	}
}
