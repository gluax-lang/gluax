package codegen

func fastLocalsHeaders(cg *Codegen) {
	cg.ln("--[[fast access locals]]")
	cg.ln("local string, table, bit, math = string, table, bit, math;")
	cg.ln("local bit_bor, bit_band, bit_bxor, bit_lshift, bit_rshift, bit_bnot = bit.bor, bit.band, bit.bxor, bit.lshift, bit.rshift, bit.bnot;")
	cg.ln("local type = type;")
	cg.ln("local pairs = pairs;")
	cg.ln("local tostring, tonumber = tostring, tonumber;")
	cg.ln("local setmetatable, getmetatable = setmetatable, getmetatable;")
	cg.ln("local SERVER, CLIENT = SERVER, CLIENT;")
}

func publicHeaders(cg *Codegen) {
	cg.ln("--[[public symbols]]")
	cg.ln("local %s = {};", PUBLIC_TBL)
}
