open Mirage
let start =
  main "Unikernel.Client"
    ~runtime_args: [ 
      runtime_arg ~pos:__POS__ "Unikernel.connect_id";
      runtime_arg ~pos:__POS__ "Unikernel.uri";
    ]
    ~packages: [
      package "cohttp-mirage";
      package "duration";
      package "yojson";
    ]
    (resolver @-> conduit @-> job)

let () =
  let stack = generic_stackv4v6 default_network in
  let res_dns = resolver_dns stack in
  let conduit = conduit_direct ~tls:true stack in
  register "ws-handler" [ start $ res_dns $ conduit ]
