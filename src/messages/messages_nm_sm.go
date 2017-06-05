package messages

type ServiceRequest struct {
	Payload  []byte
	Sequence uint32
}

type ServiceResponse struct {
	Payload  []byte
	Sequence uint32
}

type AppRegistrationRequest struct {
	ServiceInfo ServiceInfo
}

type AppRegistrationResponse struct {
	Ok    bool
	Error string
}
