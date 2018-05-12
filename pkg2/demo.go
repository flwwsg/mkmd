package pkg2

// Demo1000 struct to test
type Demo1000 struct {
	Name   string
	RID    string `dcapi:"req; def:rid123456;"`
	ID     string `dcapi:"req; alias:id;" json:"-"`
	Number int    `dcapi:"resp"`
}
