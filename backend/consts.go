package codegen

import (
	"github.com/gluax-lang/gluax/frontend"
)

const PREFIX = frontend.PreservedPrefix
const TEMP_PREFIX = PREFIX + "t_%d"
const CONTINUE_PREFIX = PREFIX + "continue_"
const BREAK_PREFIX = PREFIX + "break_"
const FUNC_PREFIX = PREFIX + "func_"
const STRUCT_PREFIX = PREFIX + "struct_"
const UNREACHABLE_PREFIX = PREFIX + "unreachable_"
const LOCAL_PREFIX = PREFIX + "local_"

const IMPORTS_TBL = PREFIX + "imports"
const PUBLIC_TBL = PREFIX + "public"

const RUN_IMPORT = PREFIX + "run_import"

const STRUCT_MARKER_PREFIX = PREFIX + "marker_struct"
