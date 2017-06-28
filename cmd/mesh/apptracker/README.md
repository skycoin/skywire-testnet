run:

go run run_apptracker.go listen_address app_id_for_ack message_id_for_ack

listen_address - host:port on which apptracker will listen for incoming messages from nodemanager/orchestration server
app_id_for_ack - message id for ack, produced by viscript. Will be the same for every message to the app. The ack from the created node will be sent with this id so viscript will know for which app it received the ack.
message_id_for_ack - message id for ack, produced by viscript. Will be the different for every message. The ack from the created node will be sent with this id so viscript will know for which messages it received the ack.

For example:
go run run_apptracker.go 127.0.0.1:20000 3 114
