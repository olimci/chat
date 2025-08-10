package main

import "github.com/lucasb-eyer/go-colorful"

type ClientDataExternal struct {
	Nick    string
	Color   string
	OPLevel OPLevel
}

func NewClientDataExternal(creator bool) ClientDataExternal {
	level := OPLevelUser

	if creator {
		level = OPLevelAdmin
	}

	return ClientDataExternal{
		Color:   colorful.HappyColor().Hex(),
		OPLevel: level,
	}
}

type ClientDataInternal struct {
	Nick string
}
