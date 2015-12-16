package tasks

import "strings"

// Marathon Task Id
// Usually in the form of AppId.uuid with '/' replaced with '_'
type Id string

func (id Id) String() string {
	return string(id)
}

// Marathon Application Id (aka PathId)
// Usually in the form of /rootGroup/subGroup/subSubGroup/name
// allowed characters: lowercase letters, digits, hyphens, slash
type AppId string

func (id AppId) String() string {
	return string(id)
}

func (id AppId) ConsulServiceName() string {
	return strings.Replace(strings.Trim(id.String(), "/"), "/", ".", -1)
}
