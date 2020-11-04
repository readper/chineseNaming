package models

type Naming struct {
    Id int64
}

type UnwantWord struct {
    NamingId int64 `xorm:"bigint(20) notnull index(pk)"`
    WordId int64 `xorm:"bigint(20)"`
    StrokeCount int64 `xorm:"bigint(20) notnull index(pk)"`
}

type Word struct {
    Id int64
    Word string `xorm:"varchar(4) notnull"`
    Bopomofo string `xorm:"varchar(40) notnull"`
    Meaning string `xorm:"varchar(500) notnull"`
    StrokeCount int64 `xorm:"bigint(20) notnull index"`
    Unwant bool `xorm:"-"`
}

type Definition struct {
    Def string `json:"def"`
    Link []string `json:"link"`
    Type string `json:"type"`
}

type Heteronym struct {
    Bopomofo string `json:"bopomofo"`
    Definitions *[]Definition `json:"definitions"`

}

type Dict struct {
    Title string `json:"title"`
    Heteronyms *[]Heteronym `json:"heteronyms"`
    StrokeCount int64 `json:"stroke_count"`
}
