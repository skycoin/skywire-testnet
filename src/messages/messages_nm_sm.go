package messages

type ServiceRegistrationRequest struct {
	ServiceInfo ServiceInfo
	Sequence    uint32
}

type ServiceRequest struct {
	RequestType  string // "by_name", "by_type", "all"
	RequestParam string // type or service name, if all then equals ""
	Sequence     uint32
}

type ServiceRegistrationResponse struct {
	Ok    bool
	Error string
}

type ServiceResponse struct {
	Services []ServiceInfo
	Error    string
}
