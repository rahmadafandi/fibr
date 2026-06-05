// Copyright 2026 Rahmad Afandi. MIT License.

package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ModuleData is the render context for module templates.
type ModuleData struct {
	Module string // go.mod module import path
	Layout string // "ddd" | "layered"
	Type   string // exported type name, e.g. "Product"
	Pkg    string // package/var name, e.g. "product"
	Plural string // table + route base, e.g. "products"
}

var moduleNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// deriveModuleNames validates name and derives the Type/Pkg/Plural forms.
// Pluralization is naive (append "s"); rename in the generated files if needed.
func deriveModuleNames(name string) (ModuleData, error) {
	if !moduleNameRe.MatchString(name) {
		return ModuleData{}, fmt.Errorf("module name %q must be a Go identifier (letters, digits, underscore; not starting with a digit)", name)
	}
	pkg := strings.ToLower(name)
	typ := strings.ToUpper(pkg[:1]) + pkg[1:]
	return ModuleData{Type: typ, Pkg: pkg, Plural: pkg + "s"}, nil
}

// planModule returns the template->dest specs for a module in the given layout.
func planModule(md ModuleData) []fileSpec {
	switch md.Layout {
	case "ddd":
		return []fileSpec{
			{"module/ddd/domain.tmpl", "internal/domain/" + md.Pkg + "/" + md.Pkg + ".go"},
			{"module/ddd/repository_iface.tmpl", "internal/domain/" + md.Pkg + "/repository.go"},
			{"module/ddd/service.tmpl", "internal/application/" + md.Pkg + "/service.go"},
			{"module/ddd/persistence.tmpl", "internal/infrastructure/persistence/" + md.Pkg + "_repository_bun.go"},
			{"module/ddd/handler.tmpl", "internal/interface/http/" + md.Pkg + "_handler.go"},
			{"module/ddd/module.tmpl", "internal/interface/http/" + md.Pkg + "_module.go"},
		}
	case "layered":
		return []fileSpec{
			{"module/layered/model.tmpl", "internal/model/" + md.Pkg + ".go"},
			{"module/layered/repository.tmpl", "internal/repository/" + md.Pkg + "_repo.go"},
			{"module/layered/service.tmpl", "internal/service/" + md.Pkg + "_service.go"},
			{"module/layered/handler.tmpl", "internal/handler/" + md.Pkg + "_handler.go"},
			{"module/layered/module.tmpl", "internal/handler/" + md.Pkg + "_module.go"},
		}
	}
	return nil
}
