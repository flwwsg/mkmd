//pkg2 login
package pkg2

//DemoLoginParams demo login
type DemoLoginParams struct {
	//jwt doc
	Jwt string `valid:"required"`
	//xx doc
	DeviceType string `valid:"required"`
	//DeviceOS doc
	DeviceOS string `valid:"required"`
	//RetailID doc
	RetailId string `valid:"required"`
}
type name string

//Resp login
type DemoLoginResp struct {
	Role       Role
	SystemTime int64
}

//Role Doc
type Role struct {
	//role name
	Name string
	Age  int
}
