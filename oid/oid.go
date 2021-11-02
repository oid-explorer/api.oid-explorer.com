package oid

type OID struct {
	Name        string `json:"name" db:"name"`
	OID         string `json:"oid" db:"oid"`
	*Properties `json:",inline,omitempty"`
}

type Properties struct {
	ObjectType   string        `json:"object_type"  db:"objectType"`
	Descriptions []Description `json:"descriptions" db:"descriptions"`
	Parent       *OID          `json:"parent" db:"parent"`
}

type Description struct {
	MIB         string `json:"mib" db:"mib"`
	Description string `json:"description" db:"description"`
}

type Relation struct {
	OID      OID        `json:"oid"`
	Children []Relation `json:"children"`
}
