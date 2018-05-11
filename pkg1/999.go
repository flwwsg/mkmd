package pkg1

// Demo struct to test
type Demo999 struct {
	ID     string `dcapi:"req; alias:id;" json:"-"`
	Number int    `dcapi:"resp"`
}
