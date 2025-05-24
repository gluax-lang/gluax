package codegen

import (
	"github.com/gluax-lang/gluax/frontend"
)

const PREFIX = frontend.PreservedPrefix
const TEMP_PREFIX = PREFIX + "_t_%d"
const CONTINUE_PREFIX = PREFIX + "_continue_"
const BREAK_PREFIX = PREFIX + "_break_"
const FUNC_PREFIX = PREFIX + "_func_"
const STRUCT_PREFIX = PREFIX + "_struct_"
const UNREACHABLE_PREFIX = PREFIX + "_unreachable_"
const LOCAL_PREFIX = PREFIX + "_local_"

const IMPORTS_TBL = PREFIX + "_imports"
const PUBLIC_TBL = PREFIX + "_public"

const RUN_IMPORT = PREFIX + "_run_import"

const STRUCT_NEW = PREFIX + "_structnew"
const STRUCT_OBJ_FIELDS = PREFIX + "_structobj_fields"
const STRUCT_OBJ_INSTANCES = PREFIX + "_structobj_instances"
