# varnish.vcl
vcl 4.0;

import std;

backend index {
    .host = "nginx";
    .port = "80";
}

backend components {
    .host = "components";
    .port = "80";
}

sub vcl_recv {
    if (req.url ~ "^/components") {
       set req.backend_hint = components;
       set req.url = regsub(req.url, "^/components", "");
       set req.ttl = 30s;
    } 
}

sub vcl_backend_response {
    set beresp.do_esi = true;
}