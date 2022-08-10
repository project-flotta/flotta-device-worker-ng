package entity

type MessageKind int

const (
	WorkloadConfigurationMessage MessageKind = iota
	ProfileConfigurationMessage
)

type Message struct {
	Kind    MessageKind
	Payload interface{}
}
