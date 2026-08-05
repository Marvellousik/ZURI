package graph

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

type CodeParser struct{}

func NewCodeParser() *CodeParser {
	return &CodeParser{}
}

// ParseFile parses a source code file (Go, TypeScript, or Python) and extracts universal structural nodes and edges (§17.3, §17.4).
func (p *CodeParser) ParseFile(repoID string, filePath string, content string) ([]GraphNode, []GraphEdge, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return p.parseGoFile(repoID, filePath, content)
	case ".ts", ".tsx", ".js", ".jsx":
		return p.parseTypeScriptFile(repoID, filePath, content)
	case ".py":
		return p.parsePythonFile(repoID, filePath, content)
	default:
		return p.parseGenericFile(repoID, filePath, content)
	}
}

func (p *CodeParser) parseGoFile(repoID string, filePath string, content string) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	var edges []GraphEdge

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		// Fallback to generic parsing if syntactic parse fails
		return p.parseGenericFile(repoID, filePath, content)
	}

	moduleName := fileNode.Name.Name
	moduleID := fmt.Sprintf("%s:%s:module:%s", repoID, filePath, moduleName)
	moduleNode := GraphNode{
		ID:         moduleID,
		RepoID:     repoID,
		Kind:       NodeKindModule,
		Name:       moduleName,
		FilePath:   filePath,
		StartLine:  1,
		EndLine:    fset.Position(fileNode.End()).Line,
		Language:   "go",
		Properties: map[string]any{"package": moduleName},
	}
	nodes = append(nodes, moduleNode)

	// Imports
	for _, imp := range fileNode.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		targetID := fmt.Sprintf("%s:import:%s", repoID, impPath)
		targetNode := GraphNode{
			ID:         targetID,
			RepoID:     repoID,
			Kind:       NodeKindModule,
			Name:       impPath,
			FilePath:   impPath,
			Language:   "go",
			Properties: map[string]any{"external": true},
		}
		nodes = append(nodes, targetNode)
		edges = append(edges, GraphEdge{
			RepoID:   repoID,
			SourceID: moduleID,
			TargetID: targetID,
			EdgeKind: EdgeKindImports,
		})
	}

	// Functions, Structs, Endpoints
	httpRouteRegex := regexp.MustCompile(`(?:HandleFunc|Handle|Get|Post|Put|Delete)\(\s*"([^"]+)"`)

	ast.Inspect(fileNode, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			funcName := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := star.X.(*ast.Ident); ok {
						funcName = ident.Name + "." + funcName
					}
				} else if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
					funcName = ident.Name + "." + funcName
				}
			}
			startLine := fset.Position(fn.Pos()).Line
			endLine := fset.Position(fn.End()).Line

			funcID := fmt.Sprintf("%s:%s:func:%s", repoID, filePath, funcName)
			funcNode := GraphNode{
				ID:        funcID,
				RepoID:    repoID,
				Kind:      NodeKindFunction,
				Name:      funcName,
				FilePath:  filePath,
				StartLine: startLine,
				EndLine:   endLine,
				Language:  "go",
			}
			nodes = append(nodes, funcNode)
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: funcID,
				EdgeKind: EdgeKindContains,
			})

			// Check body for function calls & HTTP route definitions
			if fn.Body != nil {
				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					if call, ok := inner.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							targetFunc := sel.Sel.Name
							targetID := fmt.Sprintf("%s:call:%s", repoID, targetFunc)
							edges = append(edges, GraphEdge{
								RepoID:   repoID,
								SourceID: funcID,
								TargetID: targetID,
								EdgeKind: EdgeKindCalls,
							})
						}
					}
					return true
				})
			}
		}
		return true
	})

	// Check content for HTTP endpoint declarations
	matches := httpRouteRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			route := match[1]
			endpointID := fmt.Sprintf("%s:api:%s", repoID, route)
			endpointNode := GraphNode{
				ID:         endpointID,
				RepoID:     repoID,
				Kind:       NodeKindAPIEndpoint,
				Name:       route,
				FilePath:   filePath,
				Language:   "go",
				Properties: map[string]any{"route": route},
			}
			nodes = append(nodes, endpointNode)
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: endpointID,
				EdgeKind: EdgeKindDefinesAPI,
			})
		}
	}

	return nodes, edges, nil
}

func (p *CodeParser) parseTypeScriptFile(repoID string, filePath string, content string) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	var edges []GraphEdge

	lines := strings.Split(content, "\n")
	moduleName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	moduleID := fmt.Sprintf("%s:%s:module:%s", repoID, filePath, moduleName)

	moduleNode := GraphNode{
		ID:        moduleID,
		RepoID:    repoID,
		Kind:      NodeKindModule,
		Name:      moduleName,
		FilePath:  filePath,
		StartLine: 1,
		EndLine:   len(lines),
		Language:  "typescript",
	}
	nodes = append(nodes, moduleNode)

	// Imports regex
	importRegex := regexp.MustCompile(`import\s+.*?from\s+['"]([^'"]+)['"]`)
	funcRegex := regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z0-9_]+)|(?:export\s+)?const\s+([a-zA-Z0-9_]+)\s*=\s*(?:async\s*)?\(`)
	apiRegex := regexp.MustCompile(`(?:app|router|server)\.(get|post|put|delete|patch)\(\s*['"]([^'"]+)['"]`)
	fetchRegex := regexp.MustCompile(`fetch\(\s*['"]([^'"]+)['"]`)

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1

		// Imports
		if match := importRegex.FindStringSubmatch(line); len(match) > 1 {
			impPath := match[1]
			targetID := fmt.Sprintf("%s:import:%s", repoID, impPath)
			nodes = append(nodes, GraphNode{
				ID:       targetID,
				RepoID:   repoID,
				Kind:     NodeKindModule,
				Name:     impPath,
				FilePath: impPath,
				Language: "typescript",
			})
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: targetID,
				EdgeKind: EdgeKindImports,
			})
		}

		// Functions
		if match := funcRegex.FindStringSubmatch(line); len(match) > 1 {
			fnName := match[1]
			if fnName == "" && len(match) > 2 {
				fnName = match[2]
			}
			if fnName != "" {
				funcID := fmt.Sprintf("%s:%s:func:%s", repoID, filePath, fnName)
				nodes = append(nodes, GraphNode{
					ID:        funcID,
					RepoID:    repoID,
					Kind:      NodeKindFunction,
					Name:      fnName,
					FilePath:  filePath,
					StartLine: lineNum,
					EndLine:   lineNum + 10,
					Language:  "typescript",
				})
				edges = append(edges, GraphEdge{
					RepoID:   repoID,
					SourceID: moduleID,
					TargetID: funcID,
					EdgeKind: EdgeKindContains,
				})
			}
		}

		// Defined API Endpoints
		if match := apiRegex.FindStringSubmatch(line); len(match) > 2 {
			method := strings.ToUpper(match[1])
			route := match[2]
			apiID := fmt.Sprintf("%s:api:%s:%s", repoID, method, route)
			nodes = append(nodes, GraphNode{
				ID:         apiID,
				RepoID:     repoID,
				Kind:       NodeKindAPIEndpoint,
				Name:       method + " " + route,
				FilePath:   filePath,
				Language:   "typescript",
				Properties: map[string]any{"method": method, "route": route},
			})
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: apiID,
				EdgeKind: EdgeKindDefinesAPI,
			})
		}

		// Invoked API Endpoints
		if match := fetchRegex.FindStringSubmatch(line); len(match) > 1 {
			targetURL := match[1]
			apiID := fmt.Sprintf("%s:invoked_api:%s", repoID, targetURL)
			nodes = append(nodes, GraphNode{
				ID:         apiID,
				RepoID:     repoID,
				Kind:       NodeKindAPIEndpoint,
				Name:       targetURL,
				FilePath:   filePath,
				Language:   "typescript",
				Properties: map[string]any{"url": targetURL},
			})
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: apiID,
				EdgeKind: EdgeKindInvokesAPI,
			})
		}
	}

	return nodes, edges, nil
}

func (p *CodeParser) parsePythonFile(repoID string, filePath string, content string) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	var edges []GraphEdge

	lines := strings.Split(content, "\n")
	moduleName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	moduleID := fmt.Sprintf("%s:%s:module:%s", repoID, filePath, moduleName)

	moduleNode := GraphNode{
		ID:        moduleID,
		RepoID:    repoID,
		Kind:      NodeKindModule,
		Name:      moduleName,
		FilePath:  filePath,
		StartLine: 1,
		EndLine:   len(lines),
		Language:  "python",
	}
	nodes = append(nodes, moduleNode)

	importRegex := regexp.MustCompile(`(?:import\s+([a-zA-Z0-9_\.]+)|from\s+([a-zA-Z0-9_\.]+)\s+import)`)
	funcRegex := regexp.MustCompile(`^\s*def\s+([a-zA-Z0-9_]+)\s*\(`)
	apiRegex := regexp.MustCompile(`@(?:app|router)\.(get|post|put|delete|patch)\(\s*['"]([^'"]+)['"]`)

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1

		// Imports
		if match := importRegex.FindStringSubmatch(line); len(match) > 1 {
			imp := match[1]
			if imp == "" && len(match) > 2 {
				imp = match[2]
			}
			if imp != "" {
				targetID := fmt.Sprintf("%s:import:%s", repoID, imp)
				nodes = append(nodes, GraphNode{
					ID:       targetID,
					RepoID:   repoID,
					Kind:     NodeKindModule,
					Name:     imp,
					FilePath: imp,
					Language: "python",
				})
				edges = append(edges, GraphEdge{
					RepoID:   repoID,
					SourceID: moduleID,
					TargetID: targetID,
					EdgeKind: EdgeKindImports,
				})
			}
		}

		// Functions
		if match := funcRegex.FindStringSubmatch(line); len(match) > 1 {
			fnName := match[1]
			funcID := fmt.Sprintf("%s:%s:func:%s", repoID, filePath, fnName)
			nodes = append(nodes, GraphNode{
				ID:        funcID,
				RepoID:    repoID,
				Kind:      NodeKindFunction,
				Name:      fnName,
				FilePath:  filePath,
				StartLine: lineNum,
				EndLine:   lineNum + 10,
				Language:  "python",
			})
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: funcID,
				EdgeKind: EdgeKindContains,
			})
		}

		// FastAPI/Flask Endpoints
		if match := apiRegex.FindStringSubmatch(line); len(match) > 2 {
			method := strings.ToUpper(match[1])
			route := match[2]
			apiID := fmt.Sprintf("%s:api:%s:%s", repoID, method, route)
			nodes = append(nodes, GraphNode{
				ID:         apiID,
				RepoID:     repoID,
				Kind:       NodeKindAPIEndpoint,
				Name:       method + " " + route,
				FilePath:   filePath,
				Language:   "python",
				Properties: map[string]any{"method": method, "route": route},
			})
			edges = append(edges, GraphEdge{
				RepoID:   repoID,
				SourceID: moduleID,
				TargetID: apiID,
				EdgeKind: EdgeKindDefinesAPI,
			})
		}
	}

	return nodes, edges, nil
}

func (p *CodeParser) parseGenericFile(repoID string, filePath string, content string) ([]GraphNode, []GraphEdge, error) {
	lines := strings.Split(content, "\n")
	moduleName := filepath.Base(filePath)
	moduleID := fmt.Sprintf("%s:%s:module:%s", repoID, filePath, moduleName)

	node := GraphNode{
		ID:        moduleID,
		RepoID:    repoID,
		Kind:      NodeKindModule,
		Name:      moduleName,
		FilePath:  filePath,
		StartLine: 1,
		EndLine:   len(lines),
		Language:  "generic",
	}

	return []GraphNode{node}, nil, nil
}
