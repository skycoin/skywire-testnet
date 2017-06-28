run:

go run run_nm.go domain_name ctrl_address app_tracker_address app_id_for_ack message_id_for_ack

domain_name - domain name for using alternative node hostnames
ctrlAddress - host:port for control messages exchange
app_tracker_address - address of apptracker which should be run before nodemanager is running
app_id_for_ack - message id for ack, produced by viscript. Will be the same for every message to the app. The ack from the created node will be sent with this id so viscript will know for which app it received the ack.
message_id_for_ack - message id for ack, produced by viscript. Will be the different for every message. The ack from the created node will be sent with this id so viscript will know for which messages it received the ack.

For example:
go run run_nm.go mysterious.network 0.0.0.0:5999 127.0.0.1:20000 3 114
