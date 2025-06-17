package sema

func (a *Analysis) FindGenericMethods(generic *SemGenericType, methodName string) []SemFunction {
	var methods []SemFunction
	for _, trait := range generic.Traits {
		ms := a.GetTraitMethods(trait, methodName)
		methods = append(methods, ms...)
	}
	return methods
}
