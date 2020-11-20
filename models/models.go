package models

type Naming struct {
	Id int64
}

type Order struct {
	NamingId    int64 `xorm:"bigint(20) notnull index(naming_order)"`
	Order       int64 `xorm:"bigint(20) index(naming_order)"`
	StrokeCount int64 `xorm:"bigint(20) notnull"`
}

type UnwantWord struct {
	NamingId    int64 `xorm:"bigint(20) notnull index(pk)"`
	WordId      int64 `xorm:"bigint(20)"`
	StrokeCount int64 `xorm:"bigint(20) notnull index(pk)"`
}

type UnwantName struct {
	NamingId int64  `xorm:"bigint(20) notnull index(pk)"`
	NameId   string `xorm:"varchar(30) notnull index(pk)"`
}

type Name struct {
	Id     string
	Words  []Word
	Unwant bool
}

type Word struct {
	Id          int64
	Word        string `xorm:"varchar(4) notnull"`
	Bopomofo    string `xorm:"varchar(40) notnull"`
	Meaning     string `xorm:"varchar(500) notnull"`
	StrokeCount int64  `xorm:"bigint(20) notnull index"`
	Unwant      bool   `xorm:"-"`
}

// Following for json parsing

type Dict struct {
	Title       string       `json:"title"`
	Heteronyms  *[]Heteronym `json:"heteronyms"`
	StrokeCount int64        `json:"stroke_count"`
}

type Heteronym struct {
	Bopomofo    string        `json:"bopomofo"`
	Definitions *[]Definition `json:"definitions"`
}

type Definition struct {
	Def  string   `json:"def"`
	Link []string `json:"link"`
	Type string   `json:"type"`
}
