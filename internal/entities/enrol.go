package entities

type EnrolementInfo struct {
	// features
	Features EnrolmentInfoFeatures

	// target namespace
	TargetNamespace string
}

type EnrolmentInfoFeatures struct {
	// hardware
	Hardware HardwareInfo

	// model name
	ModelName string
}
