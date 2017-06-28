run vpn client:

go run run_vpn_client.go app_id node_address proxy_port app_id_for_ack message_id_for_ack

app_id - text name of app, must be unique
node_address - node host:port which app will be talk with
proxy_port - port which vpn will be listen for web app (e.g. browser) incoming messages
app_id_for_ack - message id for ack, produced by viscript. Will be the same for every message to the app. The ack from the created node will be sent with this id so viscript will know for which app it received the ack.
message_id_for_ack - message id for ack, produced by viscript. Will be the different for every message. The ack from the created node will be sent with this id so viscript will know for which messages it received the ack.

For example:
go run run_vpn_client.go vpncli0 101.202.34.56:15000 8000 3 114


run vpn server:

go run run_vpn_server.go app_id node_address proxy_port app_id_for_ack message_id_for_ack

app_id - text name of app, must be unique
node_address - node host:port which app will be talk with
app_id_for_ack - message id for ack, produced by viscript. Will be the same for every message to the app. The ack from the created node will be sent with this id so viscript will know for which app it received the ack.
message_id_for_ack - message id for ack, produced by viscript. Will be the different for every message. The ack from the created node will be sent with this id so viscript will know for which messages it received the ack.

For example:
go run run_vpn_server.go vpnsrv0 101.202.34.56:15000 3 114
