package config

type GlobalConfig struct {
	Rc   string      `json:"id"`
	Dir  string      `json:"uri"`
	Init interface{} `json:"init"`
}
