let get_configs _ =
  let open Yojson.Basic.Util in
  Ok (`Assoc [
    "width", `Int 1024;
    "height", `Int 1024;
  ])

let handlers = [
  "get_configs", get_configs;
]
