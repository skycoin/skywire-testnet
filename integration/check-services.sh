
echo -e "\n\n" MESSAGING-DISCOVERY 

echo -e "\n\n" $MSGD/messaging-discovery/available_servers"\n"
curl $MSGD/messaging-discovery/available_servers

echo -e "\n\n" $MSGD/messaging-discovery/entry/PK_A"\n"
curl $MSGD/messaging-discovery/entry/$PK_A

echo -e "\n\n" $MSGD/messaging-discovery/entry/PK_B"\n"
curl $MSGD/messaging-discovery/entry/$PK_B

echo -e "\n\n" $MSGD/messaging-discovery/entry/PK_C"\n"
curl $MSGD/messaging-discovery/entry/$PK_C


echo -e "\n\n" TRANSPORT-DISCOVERY 

echo -e "\n\n" $TRD/security/nonces/PK_A"\n"
curl $TRD/security/nonces/$PK_A
echo -e "\n\n" $TRD/transports/edge:PK_A "\n"
curl $TRD/transports/edge:$PK_A

echo -e "\n\n" $TRD/security/nonces/PK_B"\n"
curl $TRD/security/nonces/$PK_B
echo -e "\n\n" $TRD/transports/edge:PK_B "\n"
curl $TRD/transports/edge:$PK_B
echo -e "\n\n" $TRD/security/nonces/PK_C"\n"
curl $TRD/security/nonces/$PK_C
echo -e "\n\n" $TRD/transports/edge:PK_C "\n"
curl $TRD/transports/edge:$PK_C

echo -e "\n\n" ROUTE-FINDER "\n\n"

echo 
echo '{"src_pk":''"'$PK_A'","dst_pk":''"'$PK_C'","min_hops":0, "max_hops":50}'
echo '{"src_pk":''"'$PK_A'","dst_pk":''"'$PK_C'","min_hops":0, "max_hops":50}'  |curl -X GET $RF/routes -d@-
echo
echo '{"src_pk":''"'$PK_A'","dst_pk":''"'$PK_B'","min_hops":0, "max_hops":50}'
echo '{"src_pk":''"'$PK_A'","dst_pk":''"'$PK_B'","min_hops":0, "max_hops":50}'  |curl -X GET $RF/routes -d@-
echo 
echo '{"src_pk":''"'$PK_B'","dst_pk":''"'$PK_C'","min_hops":0, "max_hops":50}'
echo '{"src_pk":''"'$PK_B'","dst_pk":''"'$PK_C'","min_hops":0, "max_hops":50}'  |curl -X GET $RF/routes -d@-
