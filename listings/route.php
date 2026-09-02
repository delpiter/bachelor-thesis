function getCountryByRepl($lan_ip)
{
    $db = cnDBOpen();

    $qry = "SELECT label_desc, repl_code, country, HiddenTable.component_value_json, public_ipv4 from HiddenTable2
            join HiddenTable3 on repl_code=host_repl_code
            JOIN HiddenTable on HiddenTable2.country=HiddenTable.report_field
            where HiddenTable.component_code='maps_country' and is_main=true and is_powerdns=true
            and port='xx' and is_alias=false
            and host_ip=$1";

    $res = pg_query_params($db, $qry, array($lan_ip));
    $nRows = pg_num_rows($res);

    $qry2 = "SELECT HiddenTable4.name as label_desc, HiddenTable5.country as country from HiddenTable6
             join HiddenTable4 on HiddenTable6.cluster=HiddenTable4.id
             join HiddenTable5 on HiddenTable4.location=HiddenTable5.id
             where HiddenTable6.lan_ip=$1";

    $res2 = pg_query_params($db, $qry2, array($lan_ip));
    $nRows2 = pg_num_rows($res2);
    // ...

    if ($nRows > 0) {
        for ($i = 0; $i < $nRows; $i++) {
            $row = pg_fetch_array($res);
            $json = json_decode($row["location_json"]);
            pg_close($db);
            return array(...);
        }
    }
}
