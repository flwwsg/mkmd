//pkg2 login2
package pkg2

//Demo2LoginParams demo2login
type Demo2LoginParams struct {
	//jwt doc
	Jwt string `valid:"required"`
	//xx doc
}

//Resp login
type Demo2LoginResp struct {
	Role Role
}
