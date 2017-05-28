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

type AppListRequest struct {
	RequestType  string // "by_name", "by_type", "all"
	RequestParam string // type or service name, if all then equals ""
}

type AppListResponse struct {
	Apps []ServiceInfo
}
