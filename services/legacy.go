package services

type LegacyMarshaler interface {
	Unmarshal([]byte, *ServiceConfig) error
}

var legacyUnmarshalers []LegacyMarshaler

func RegisterLegacyMarshaler(l LegacyMarshaler) {
	legacyUnmarshalers = append(legacyUnmarshalers, l)
}
