# Skywire SOCKS5 proxy client app

`socksproxy-client` app implements client for the SOCKS5 app.

It opens persistent `skywire` connection to the configured remote visor
and local TCP port, all incoming TCP traffics is forwarded to the
~skywire~ connection.

Any conventional SOCKS5 client should be able to connect to the proxy client.

Please check docs for `socksproxy` app for further instructions.
