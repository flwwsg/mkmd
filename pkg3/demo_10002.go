package pkg3

//接口功能说明如：登录
type LoginParams struct {
	//jwt描述
	Jwt string `valid:"required"` //jwt描述2, 优先级更高
	//user id 描述
	UserId string `valid:"required"`
	//device id 描述
	DeviceId string `valid:"required"`
	//device type 描述
	DeviceType string `valid:"required"`
	//device os 描述
	DeviceOS string `valid:"required"`
	//retail id 描述
	RetailId string `valid:"required"`
	//play second 描述
	PlaySecond int `valid:"required"`
}

//接口功能说明(可选)如：登录
type LoginResp struct {
	//role 描述
	Role Role
	//system time 描述
	SystemTime int64
}

type Role struct {
	Id       string
	ShowId   int
	Nickname string
	Sex      int16
	Avatar   int16
	Lv       int16
	VipLv    int16
	Exp      int
	Gold     int64
}
