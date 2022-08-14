package view

type EVMCall struct {
	Address string `json:"address" bson:"address"`
	Data    string `json:"data" bson:"data"`
}
