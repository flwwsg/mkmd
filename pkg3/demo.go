package pkg3

// Demo2000 specify inner struct
type Demo2000 struct {
	FamilyID  string `dcapi:"req; alias:fid"`
	Demo      string `dcapi:"resp"`
	ChildInfo Child  `dcapi:"resp"`
}

// Child child info
type Child struct {
	CID     string `dcapi:"resp"`
	Name    string `dcapi:"resp"`
	YearOld Age    `dcapi:"resp"`
	Sex     int    `dcapi:"resp; def:0"`
}

type Age struct {
	Year string `dcapi:"resp"`
	Day  string `dcapi:"resp"`
}
