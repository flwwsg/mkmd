package pkg1

// Demo999 struct to test
type Demo999Resp struct {
	ID     string `dcapi:"req; alias:id;" json:"-"`
	Number int    `dcapi:"resp"`
	Demo   int64  `dcapi:"resp"`
	RID    string `dcapi:"req; def:rid12"`
	ID2    string `dcapi:"resp"`
}
