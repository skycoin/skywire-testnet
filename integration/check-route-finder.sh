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
