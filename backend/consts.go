package codegen

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend"
)

var definedConsts = make(map[string]bool)

func defineConst(suffix string) string {
	for existing := range definedConsts {
		if strings.HasPrefix(suffix, existing) {
			panic(fmt.Sprintf("constant suffix %q starts with existing suffix %q", suffix, existing))
		}
		if strings.HasPrefix(existing, suffix) {
			panic(fmt.Sprintf("existing suffix %q starts with new suffix %q", existing, suffix))
		}
	}
	definedConsts[suffix] = true
	return frontend.PreservedPrefix + suffix
}

var TEMP_PREFIX = defineConst("temp_%d")
var CONTINUE_PREFIX = defineConst("continue_")
var BREAK_PREFIX = defineConst("break_")
var RETURN_PREFIX = defineConst("return_")
var FUNC_PREFIX = defineConst("func_")
var CLASS_PREFIX = defineConst("class_")
var TRAIT_PREFIX = defineConst("trait_")
var UNREACHABLE_PREFIX = defineConst("unreachable_")
var LOCAL_PREFIX = defineConst("local_")
var DYNTRAIT_PREFIX = defineConst("dyn_trait_")

var IMPORTS_TBL = defineConst("imports")
var PUBLIC_TBL = defineConst("public")

var RUN_IMPORT = defineConst("run_import")

var CLASS_MARKER_PREFIX = defineConst("marker_class")
