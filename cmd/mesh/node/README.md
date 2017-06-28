run:

go run run_node.go nodeAddress nodemanager_address need_connect port_for_app_talk hostname app_id_for_ack message_id_for_ack

nodeAddress - node host:port for control messages exchange
nodemanager_address - nodemanager external address for control messages exchange
need_connect - if node needs to be connected randomly
port_for_app_talk - tcp port at which node will listen messages from apps (host will be the same as in nodeAddress)
hostname - alternative hostname which can be used instead of pubkey
app_id_for_ack - message id for ack, produced by viscript. Will be the same for every message to the app. The ack from the created node will be sent with this id so viscript will know for which app it received the ack.
message_id_for_ack - message id for ack, produced by viscript. Will be the different for every message. The ack from the created node will be sent with this id so viscript will know for which messages it received the ack.

For example:
go run run_node.go 111.222.123.44:5000 202.101.65.43:5999 true 15000 node0 3 114
