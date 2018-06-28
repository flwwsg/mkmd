package pkg2

//DemoLoginParams struct to test
type DemoLoginParams struct {
	//jwt doc
	Jwt string `valid:"required"` // jwt comment
	//xx doc
	DeviceType string `valid:"required"`
	//DeviceOS doc
	DeviceOS string `valid:"required"`
	//RetailID doc
	RetailId string `valid:"required"`
}

type DemoLoginResp struct {
	Role       Role
	SystemTime int64
}

//Role Doc
type Role struct {
	Name string
	Age  int
}
