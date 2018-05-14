package pkg3

// Demo2001 specify inner struct
type Demo2001 struct {
	FamilyID string `dcapi:"req; alias:fid"`
	FID      []int  `dcapi:"resp; " `
	Demo     string `dcapi:"resp"`
}
