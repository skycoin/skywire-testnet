package dbus

type DbusInstance struct {
	PubsubChannels map[ChannelId]*PubsubChannel
	Resources      []ResourceMeta
}

func (di *DbusInstance) Init() {
	println("<dbus/registry>.Init()")
	di.PubsubChannels = make(map[ChannelId]*PubsubChannel)
	di.Resources = make([]ResourceMeta, 0)
}

//register that a resource exists
func (di *DbusInstance) ResourceRegister(ResourceId ResourceId, ResourceType ResourceType) {
	x := ResourceMeta{}
	x.Id = ResourceId
	x.Type = ResourceType

	di.Resources = append(di.Resources, x)
}

//remove resource from list
func (di *DbusInstance) ResourceUnregister(ResourceID ResourceId, ResourceType ResourceType) {
	println("<dbus/registry>.ResourceUnregister()")
	println("FIXME/TODO: THIS IS NOT CALLED ANYWHERE")

	for i, resourceMeta := range di.Resources {
		if resourceMeta.Id == ResourceID {
			di.Resources = append(di.Resources[:i], di.Resources[i+1:]...)
		}
	}
}
