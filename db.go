package main

type Agenda struct {
	Id             string   `json:"id" storm:"id"`
	Description    string   `json:"description"`
	Mask           uint16   `json:"mask"`
	StartTime      uint64   `json:"starttime"`
	ExpireTime     uint64   `json:"expiretime"`
	Status         string   `json:"status"`
	QuorumProgress float64  `json:"quorumprogress"`
	Choices        []Choice `json:"choices"`
}

type Choice struct {
	Id          string  `json:"id" storm:"id"`
	Description string  `json:"description"`
	Bits        uint16  `json:"bits"`
	IsIgnore    bool    `json:"isignore"`
	IsNo        bool    `json:"isno"`
	Count       uint32  `json:"count"`
	Progress    float64 `json:"progress"`
}
