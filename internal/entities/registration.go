package entities

type RegistrationInfo struct {
	// Certificate Signing Request to be signed by flotta-operator CA
	CertificateRequest string

	// hardware info
	Hardware HardwareInfo
}
